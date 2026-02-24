package config
package config

import (
	"testing"
)

func TestValidate_Valid(t *testing.T) {
	cfg := &StaticConfig{
		Port:     8080,
		LogLevel: 5,
		AppKey:   "test-key",
		BaseURL:  DefaultBaseURL,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestValidate_StdioMode(t *testing.T) {
	cfg := &StaticConfig{
		Port:     0,
		LogLevel: 5,
		AppKey:   "test-key",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("port=0 (stdio mode) should be valid, got error: %v", err)
	}
}

func TestValidate_MissingAppKey(t *testing.T) {
	cfg := &StaticConfig{
		Port:     8080,
		LogLevel: 5,
		AppKey:   "",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing app_key")
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"negative", -1},
		{"too_high", 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &StaticConfig{
				Port:     tt.port,
				LogLevel: 5,
				AppKey:   "test-key",
			}
			if err := cfg.Validate(); err == nil {
				t.Errorf("expected error for port %d", tt.port)
			}
		})
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level int
	}{
		{"negative", -1},
		{"too_high", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &StaticConfig{
				Port:     8080,
				LogLevel: tt.level,
				AppKey:   "test-key",
			}
			if err := cfg.Validate(); err == nil {
				t.Errorf("expected error for log_level %d", tt.level)
			}
		})
	}
}

func TestGetPortString(t *testing.T) {
	tests := []struct {
		name string
		port int
		want string
	}{
		{"stdio_mode", 0, ""},
		{"http_mode", 8080, ":8080"},
		{"custom_port", 3000, ":3000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &StaticConfig{Port: tt.port}
			if got := cfg.GetPortString(); got != tt.want {
				t.Errorf("GetPortString() = %q, want %q", got, tt.want)
			}
		})
	}
}
