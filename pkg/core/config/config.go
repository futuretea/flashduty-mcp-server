package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	// DefaultBaseURL is the default FlashDuty API base URL
	DefaultBaseURL = "https://api.flashcat.cloud"
)

// StaticConfig represents the static configuration for the FlashDuty MCP Server
type StaticConfig struct {
	// Server configuration
	Port int `mapstructure:"port"`

	SSEBaseURL string `mapstructure:"sse_base_url"`

	// Logging configuration
	LogLevel int `mapstructure:"log_level"`

	// FlashDuty API configuration
	AppKey  string `mapstructure:"app_key"`
	BaseURL string `mapstructure:"base_url"`

	// Security configuration
	ReadOnly bool `mapstructure:"read_only"`

	// Tool configuration
	EnabledTools  []string `mapstructure:"enabled_tools"`
	DisabledTools []string `mapstructure:"disabled_tools"`

	// Timezone configuration
	DefaultTimezone string `mapstructure:"default_timezone"`
}

// Validate checks that required fields are present and values are within allowed ranges.
func (c *StaticConfig) Validate() error {
	// Validate port
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535, got %d", c.Port)
	}

	// Validate log level
	if c.LogLevel < 0 || c.LogLevel > 9 {
		return fmt.Errorf("log_level must be between 0 and 9, got %d", c.LogLevel)
	}

	// Validate FlashDuty API key
	if c.AppKey == "" {
		return fmt.Errorf("app_key is required")
	}

	// Validate timezone
	if _, err := time.LoadLocation(c.DefaultTimezone); err != nil {
		return fmt.Errorf("invalid default_timezone %q: %w", c.DefaultTimezone, err)
	}

	return nil
}

// LoadConfig loads configuration from file and environment variables using Viper
// Priority: command-line flags > environment variables > config file > defaults
func LoadConfig(configPath string) (*StaticConfig, error) {
	// Use the global viper instance to access bound command-line flags
	v := viper.GetViper()

	// Set defaults
	v.SetDefault("base_url", DefaultBaseURL)
	v.SetDefault("default_timezone", "Asia/Shanghai")

	// Set configuration file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		v.SetConfigType("yaml")
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Configure environment variable support
	// Environment variables use FLASHDUTY_MCP_ prefix and replace - with _
	v.SetEnvPrefix("FLASHDUTY_MCP")
	v.AllowEmptyEnv(true)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.AutomaticEnv()

	// Unmarshal configuration into struct
	config := &StaticConfig{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// GetPortString returns the port as a string in the format ":port"
func (c *StaticConfig) GetPortString() string {
	if c.Port == 0 {
		return ""
	}
	return fmt.Sprintf(":%d", c.Port)
}
