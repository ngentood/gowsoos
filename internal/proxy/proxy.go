package proxy

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/pkg/errors"
	"gowsoos/internal/config"
	"gowsoos/internal/metrics"
)

const (
	defaultTimeout         = 30 * time.Second
	defaultReadBufferSize  = 32 * 1024 // 32 KB
	webSocketMagicString   = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	defaultHandshakeStatus = "101 Switching Protocols"
)

// ProxyConnection interface for network connections
type ProxyConnection interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
}

// Proxy handles the SSH proxying logic
type Proxy struct {
	config  *config.Config
	logger  *slog.Logger
	metrics *metrics.Metrics
}

// NewProxy creates a new proxy instance
func NewProxy(cfg *config.Config, logger *slog.Logger, m *metrics.Metrics) *Proxy {
	return &Proxy{
		config:  cfg,
		logger:  logger,
		metrics: m,
	}
}

// HandleConnection manages individual proxy connections
func (p *Proxy) HandleConnection(ctx context.Context, clientConn ProxyConnection, isTLSClient bool) {
	defer func() {
		clientConn.Close()
		p.metrics.RecordConnectionClosed()
	}()

	startTime := time.Now()
	connType := "http"
	if isTLSClient {
		connType = "tls"
	}

	// Perform WebSocket handshake or custom handshake
	if err := p.performHandshake(clientConn); err != nil {
		p.logger.Error("Handshake failed", "error", err)
		p.metrics.RecordError("handshake", err.Error())
		p.metrics.RecordConnection(connType, "failed")
		return
	}

	// Establish connection to destination
	destConn, err := net.DialTimeout("tcp", p.config.DstAddress, defaultTimeout)
	if err != nil {
		p.logger.Error("Failed to connect to destination", "error", err)
		p.metrics.RecordError("destination", err.Error())
		p.metrics.RecordConnection(connType, "failed")
		return
	}
	defer destConn.Close()

	p.metrics.RecordConnection(connType, "success")

	// Handle connection based on TLS mode
	if isTLSClient && p.config.TLSMode == "stunnel" {
		// Direct stream copying for stunnel mode
		p.streamConnections(destConn, clientConn)
		p.metrics.RecordConnectionDuration(connType+"-stunnel", time.Since(startTime).Seconds())
		return
	}

	// Discard initial payload for standard mode
	if err := p.discardPayload(clientConn); err != nil {
		p.logger.Error("Failed to discard payload", "error", err)
		p.metrics.RecordError("payload", err.Error())
		return
	}

	// Stream connections
	p.streamConnections(destConn, clientConn)
	p.metrics.RecordConnectionDuration(connType, time.Since(startTime).Seconds())
}

// performHandshake handles WebSocket or custom handshake
func (p *Proxy) performHandshake(conn ProxyConnection) error {
	if p.config.HandshakeCode != "" {
		// Custom handshake response
		_, err := conn.Write([]byte(fmt.Sprintf("HTTP/1.1 %s Ok\r\n\r\n", p.config.HandshakeCode)))
		return errors.Wrap(err, "failed to write custom handshake response")
	}

	// Default WebSocket handshake
	secWebSocketKey := "Y2FmcnQ2NTRlY2Z2Z3ludTg="
	h := sha1.New()
	h.Write([]byte(secWebSocketKey + webSocketMagicString))
	secWebSocketAccept := base64.StdEncoding.EncodeToString(h.Sum(nil))

	resp := fmt.Sprintf("HTTP/1.1 %s\r\n"+
		"Upgrade: websocket\r\n"+
		"Connection: Upgrade\r\n"+
		"Sec-WebSocket-Accept: %s\r\n\r\n",
		defaultHandshakeStatus, secWebSocketAccept)

	_, err := conn.Write([]byte(resp))
	return errors.Wrap(err, "failed to write websocket handshake response")
}

// discardPayload reads and discards initial payload
func (p *Proxy) discardPayload(conn ProxyConnection) error {
	buffer := make([]byte, defaultReadBufferSize)
	_, err := io.ReadAtLeast(conn, buffer, 5)
	return errors.Wrap(err, "failed to discard initial payload")
}

// streamConnections handles bidirectional data streaming
func (p *Proxy) streamConnections(src, dst ProxyConnection) {
	errChan := make(chan error, 2)

	// Copy from src to dst
	go func() {
		bytesCopied, err := io.Copy(dst, &byteCounter{conn: src, metrics: p.metrics, direction: "src_to_dst"})
		if err != nil && err != io.EOF {
			errChan <- errors.Wrap(err, "failed to copy from src to dst")
		} else {
			p.logger.Debug("Data transfer completed", "direction", "src_to_dst", "bytes", bytesCopied)
			errChan <- nil
		}
	}()

	// Copy from dst to src
	go func() {
		bytesCopied, err := io.Copy(src, &byteCounter{conn: dst, metrics: p.metrics, direction: "dst_to_src"})
		if err != nil && err != io.EOF {
			errChan <- errors.Wrap(err, "failed to copy from dst to src")
		} else {
			p.logger.Debug("Data transfer completed", "direction", "dst_to_src", "bytes", bytesCopied)
			errChan <- nil
		}
	}()

	// Wait for first error or completion
	<-errChan
}

// byteCounter wraps a connection to count bytes transferred
type byteCounter struct {
	conn      ProxyConnection
	metrics   *metrics.Metrics
	direction string
}

func (bc *byteCounter) Read(p []byte) (int, error) {
	n, err := bc.conn.Read(p)
	if n > 0 {
		bc.metrics.RecordBytesTransferred(bc.direction, int64(n))
	}
	return n, err
}

func (bc *byteCounter) Write(p []byte) (int, error) {
	n, err := bc.conn.Write(p)
	if n > 0 {
		bc.metrics.RecordBytesTransferred(bc.direction, int64(n))
	}
	return n, err
}

// TLSConfig creates a TLS configuration for the proxy
func TLSConfig(privateKey, publicKey string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(privateKey, publicKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load TLS certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12, // Enforce modern TLS
	}, nil
}