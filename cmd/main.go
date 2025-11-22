package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"gowsoos/internal/banner"
	"gowsoos/internal/config"
	"gowsoos/internal/metrics"
	"gowsoos/internal/server"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Main is the entry point for the application
func Main() {
	var rootCmd = &cobra.Command{
		Use:   "gowsoos",
		Short: "SSH over HTTP WebSocket Proxy with SSL SNI Support",
		Long: `gowsoos is a high-performance SSH over HTTP WebSocket proxy server.
It provides secure tunneling for SSH connections through HTTP WebSocket handlers
with SSL SNI support. Up to 20 times faster than Python similar proxies.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date),
		RunE:   runProxy,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Only print banner if not showing version or help
			if !cmd.Flags().Changed("version") && !cmd.Flags().Changed("help") {
				banner.PrintBanner()
			}
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "Configuration file path (default: config.yaml)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Show version information")

	// Add configuration flags (for backward compatibility)
	rootCmd.Flags().StringP("addr", "a", ":2086", "Set port for listening clients")
	rootCmd.Flags().String("tls-addr", ":443", "Set port for listening clients if using TLS mode")
	rootCmd.Flags().String("dst-addr", "127.0.0.1:22", "Set internal IP for SSH server redirection")
	rootCmd.Flags().String("custom-handshake", "", "Set custom HTTP code for response")
	rootCmd.Flags().Bool("tls", false, "Enable TLS")
	rootCmd.Flags().String("private-key", "/etc/gowsoos/tls/private.pem", "Path to private certificate if using TLS")
	rootCmd.Flags().String("public-key", "/etc/gowsoos/tls/public.key", "Path to public certificate if using TLS")
	rootCmd.Flags().String("tls-mode", "handshake", "TLS mode: 'handshake' or 'stunnel'")
	rootCmd.Flags().Bool("metrics", false, "Enable Prometheus metrics")
	rootCmd.Flags().String("metrics-port", ":9090", "Metrics server port")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runProxy(cmd *cobra.Command, args []string) error {
	// Check for version flag
	if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
		fmt.Printf("gowsoos version %s\n", cmd.Version)
		return nil
	}

	// Load configuration
	configFile, _ := cmd.Flags().GetString("config")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override config with command-line flags
	if err := overrideConfigWithFlags(cmd, cfg); err != nil {
		return fmt.Errorf("failed to override config with flags: %w", err)
	}

	// Validate final configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Setup logger
	logLevel, _ := cmd.Flags().GetString("log-level")
	cfg.LogLevel = logLevel
	logger := setupLogger(cfg.GetLogLevel())

	// Setup metrics
	m := metrics.NewMetrics(cfg.MetricsEnabled, logger)

	// Create and start server
	srv := server.NewServer(cfg, logger, m)

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", "signal", sig)
		srv.Stop()
		cancel()
	}()

	// Start server
	if err := srv.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Wait for shutdown
	<-ctx.Done()
	srv.Wait()

	logger.Info("Server shutdown complete")
	return nil
}

func overrideConfigWithFlags(cmd *cobra.Command, cfg *config.Config) error {
	flags := []struct {
		name     string
		target   interface{}
		required bool
	}{
		{"addr", &cfg.Address, false},
		{"tls-addr", &cfg.TLSAddress, false},
		{"dst-addr", &cfg.DstAddress, false},
		{"custom-handshake", &cfg.HandshakeCode, false},
		{"tls", &cfg.TLSEnabled, false},
		{"private-key", &cfg.TLSPrivateKey, false},
		{"public-key", &cfg.TLSPublicKey, false},
		{"tls-mode", &cfg.TLSMode, false},
		{"metrics", &cfg.MetricsEnabled, false},
		{"metrics-port", &cfg.MetricsPort, false},
	}

	for _, f := range flags {
		if cmd.Flags().Changed(f.name) {
			switch t := f.target.(type) {
			case *string:
				val, err := cmd.Flags().GetString(f.name)
				if err != nil {
					return err
				}
				*t = val
			case *bool:
				val, err := cmd.Flags().GetBool(f.name)
				if err != nil {
					return err
				}
				*t = val
			}
		}
	}

	return nil
}

func setupLogger(level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Use JSON handler for production, text handler for development
	if os.Getenv("ENV") == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}