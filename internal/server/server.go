package server

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gowsoos/internal/config"
	"gowsoos/internal/metrics"
	"gowsoos/internal/proxy"
)

// Server manages HTTP and TLS servers
type Server struct {
	config    *config.Config
	logger    *slog.Logger
	metrics   *metrics.Metrics
	proxy     *proxy.Proxy
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, logger *slog.Logger, m *metrics.Metrics) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Server{
		config:  cfg,
		logger:  logger,
		metrics: m,
		proxy:   proxy.NewProxy(cfg, logger, m),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts both HTTP and TLS servers
func (s *Server) Start() error {
	serverErrChan := make(chan error, 2)

	// Start HTTP server
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.startHTTPServer(); err != nil {
			serverErrChan <- errors.Wrap(err, "HTTP server failed")
		}
	}()

	// Start TLS server if enabled
	if s.config.TLSEnabled {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := s.startTLSServer(); err != nil {
				serverErrChan <- errors.Wrap(err, "TLS server failed")
			}
		}()
	}

	// Start metrics server if enabled
	if s.config.MetricsEnabled {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := s.metrics.StartMetricsServer(s.config.MetricsPort); err != nil {
				s.logger.Error("Metrics server failed", "error", err)
			}
		}()
	}

	// Wait for any server to fail
	go func() {
		err := <-serverErrChan
		if err != nil {
			s.logger.Error("Server error", "error", err)
			s.Stop()
		}
	}()

	return nil
}

// Stop gracefully shuts down all servers
func (s *Server) Stop() {
	s.logger.Info("Shutting down servers...")
	s.cancel()
	s.wg.Wait()
	s.logger.Info("All servers stopped")
}

// Wait waits for all servers to complete
func (s *Server) Wait() {
	s.wg.Wait()
}

// startHTTPServer sets up the HTTP proxy server
func (s *Server) startHTTPServer() error {
	addr, err := net.ResolveTCPAddr("tcp", s.config.Address)
	if err != nil {
		return errors.Wrap(err, "failed to resolve TCP address")
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return errors.Wrap(err, "failed to listen on HTTP server")
	}
	defer listener.Close()

	s.logger.Info("HTTP Server listening",
		slog.String("address", s.config.Address),
		slog.String("redirect", s.config.DstAddress))

	// Setup graceful shutdown
	go func() {
		<-s.ctx.Done()
		s.logger.Info("Shutting down HTTP server...")
		listener.Close()
	}()

	// Accept connections with timeout
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
			// Set accept timeout to allow context checking
			if err := listener.SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
				s.logger.Error("Failed to set deadline", "error", err)
				continue
			}

			conn, err := listener.AcceptTCP()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Timeout is normal for context checking
				}
				s.logger.Error("Failed to accept TCP connection", "error", err)
				continue
			}

			// Configure connection
			if err := s.configureConnection(conn); err != nil {
				s.logger.Error("Failed to configure connection", "error", err)
				conn.Close()
				continue
			}

			// Handle connection
			go s.proxy.HandleConnection(s.ctx, conn, false)
		}
	}
}

// startTLSServer sets up the TLS proxy server
func (s *Server) startTLSServer() error {
	tlsConfig, err := proxy.TLSConfig(s.config.TLSPrivateKey, s.config.TLSPublicKey)
	if err != nil {
		return errors.Wrap(err, "failed to create TLS config")
	}

	listener, err := tls.Listen("tcp", s.config.TLSAddress, tlsConfig)
	if err != nil {
		return errors.Wrap(err, "failed to listen on TLS server")
	}
	defer listener.Close()

	s.logger.Info("TLS Server listening",
		slog.String("address", s.config.TLSAddress),
		slog.String("redirect", s.config.DstAddress))

	// Setup graceful shutdown
	go func() {
		<-s.ctx.Done()
		s.logger.Info("Shutting down TLS server...")
		listener.Close()
	}()

	// Accept connections
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
			conn, err := listener.Accept()
			if err != nil {
				s.logger.Error("Failed to accept TLS connection", "error", err)
				continue
			}

			// Handle connection
			go s.proxy.HandleConnection(s.ctx, conn, true)
		}
	}
}

// configureConnection configures TCP connection settings
func (s *Server) configureConnection(conn *net.TCPConn) error {
	// Enable keep-alive
	if err := conn.SetKeepAlive(s.config.KeepAlive); err != nil {
		return errors.Wrap(err, "failed to set keep-alive")
	}

	// Set keep-alive period (if supported)
	if s.config.KeepAlive {
		if err := conn.SetKeepAlivePeriod(30 * time.Second); err != nil {
			// Some systems may not support this, log but don't fail
			s.logger.Debug("Failed to set keep-alive period", "error", err)
		}
	}

	// Set no delay for better performance
	if err := conn.SetNoDelay(s.config.NoDelay); err != nil {
		return errors.Wrap(err, "failed to set no delay")
	}

	return nil
}