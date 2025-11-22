package config

import (
	"fmt"
	"log/slog"

	"github.com/spf13/viper"
)

// Config holds the configuration for the SSH proxy
type Config struct {
	Address        string `mapstructure:"address"`
	TLSAddress     string `mapstructure:"tls_address"`
	DstAddress     string `mapstructure:"dst_address"`
	HandshakeCode  string `mapstructure:"handshake_code"`
	TLSEnabled     bool   `mapstructure:"tls_enabled"`
	TLSPrivateKey  string `mapstructure:"tls_private_key"`
	TLSPublicKey   string `mapstructure:"tls_public_key"`
	TLSMode        string `mapstructure:"tls_mode"`
	ConfigFile     string `mapstructure:"config_file"`
	LogLevel       string `mapstructure:"log_level"`
	MetricsEnabled bool   `mapstructure:"metrics_enabled"`
	MetricsPort    string `mapstructure:"metrics_port"`
	
	// Security and performance settings
	MaxConnections int  `mapstructure:"max_connections"`
	Timeout        int  `mapstructure:"timeout"`
	BufferSize     int  `mapstructure:"buffer_size"`
	KeepAlive      bool `mapstructure:"keep_alive"`
	NoDelay        bool `mapstructure:"no_delay"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Address:        ":2086",
		TLSAddress:     ":443",
		DstAddress:     "127.0.0.1:22",
		HandshakeCode:  "",
		TLSEnabled:     false,
		TLSPrivateKey:  "/etc/gowsoos/tls/private.pem",
		TLSPublicKey:   "/etc/gowsoos/tls/public.key",
		TLSMode:        "handshake",
		ConfigFile:     "/etc/gowsoos/config.yaml",
		LogLevel:       "info",
		MetricsEnabled: false,
		MetricsPort:    ":9090",
		MaxConnections: 1000,
		Timeout:        30,
		BufferSize:     32768,
		KeepAlive:      true,
		NoDelay:        true,
	}
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	// Set config file path and name
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// Try FHS paths first, then fallback to local paths
		configPaths := []string{
			"/etc/gowsoos/config.yaml",
			"/etc/gowsoos/config.yml",
			"$HOME/.gowsoos/config.yaml",
			"$HOME/.gowsoos/config.yml",
			"./config.yaml",
			"./config.yml",
		}

		for _, path := range configPaths {
			viper.SetConfigFile(path)
			if err := viper.ReadInConfig(); err == nil {
				break
			}
		}

		// If no config file found, set default search paths
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("/etc/gowsoos")
		viper.AddConfigPath("$HOME/.gowsoos")
		viper.AddConfigPath(".")
	}

	// Set environment variable prefix
	viper.SetEnvPrefix("GOWSOOS")
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("address", config.Address)
	viper.SetDefault("tls_address", config.TLSAddress)
	viper.SetDefault("dst_address", config.DstAddress)
	viper.SetDefault("handshake_code", config.HandshakeCode)
	viper.SetDefault("tls_enabled", config.TLSEnabled)
	viper.SetDefault("tls_private_key", config.TLSPrivateKey)
	viper.SetDefault("tls_public_key", config.TLSPublicKey)
	viper.SetDefault("tls_mode", config.TLSMode)
	viper.SetDefault("log_level", config.LogLevel)
	viper.SetDefault("metrics_enabled", config.MetricsEnabled)
	viper.SetDefault("metrics_port", config.MetricsPort)
	viper.SetDefault("max_connections", config.MaxConnections)
	viper.SetDefault("timeout", config.Timeout)
	viper.SetDefault("buffer_size", config.BufferSize)
	viper.SetDefault("keep_alive", config.KeepAlive)
	viper.SetDefault("no_delay", config.NoDelay)

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, use defaults and environment variables
	}

	// Unmarshal config
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.TLSMode != "handshake" && c.TLSMode != "stunnel" {
		return fmt.Errorf("invalid tls_mode: %s (must be 'handshake' or 'stunnel')", c.TLSMode)
	}

	if c.TLSEnabled {
		if c.TLSPrivateKey == "" {
			return fmt.Errorf("tls_private_key is required when TLS is enabled")
		}
		if c.TLSPublicKey == "" {
			return fmt.Errorf("tls_public_key is required when TLS is enabled")
		}
	}

	if c.MaxConnections <= 0 {
		return fmt.Errorf("max_connections must be positive")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if c.BufferSize <= 0 {
		return fmt.Errorf("buffer_size must be positive")
	}

	return nil
}

// GetLogLevel returns the slog level based on configuration
func (c *Config) GetLogLevel() slog.Level {
	switch c.LogLevel {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}