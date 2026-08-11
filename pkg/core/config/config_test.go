package config

import (
	"testing"
)

func TestValidate_Valid(t *testing.T) {
	cfg := &StaticConfig{
		Port:            8080,
		LogLevel:        5,
		AppKey:          "test-key",
		BaseURL:         DefaultBaseURL,
		DefaultTimezone: "Asia/Shanghai",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestValidate_StdioMode(t *testing.T) {
	cfg := &StaticConfig{
		Port:            0,
		LogLevel:        5,
		AppKey:          "test-key",
		DefaultTimezone: "Asia/Shanghai",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("port=0 (stdio mode) should be valid, got error: %v", err)
	}
}

func TestValidate_MissingAppKey(t *testing.T) {
	cfg := &StaticConfig{
		Port:            8080,
		LogLevel:        5,
		AppKey:          "",
		DefaultTimezone: "Asia/Shanghai",
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
				Port:            tt.port,
				LogLevel:        5,
				AppKey:          "test-key",
				DefaultTimezone: "Asia/Shanghai",
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
				Port:            8080,
				LogLevel:        tt.level,
				AppKey:          "test-key",
				DefaultTimezone: "Asia/Shanghai",
			}
			if err := cfg.Validate(); err == nil {
				t.Errorf("expected error for log_level %d", tt.level)
			}
		})
	}
}

func TestValidate_ValidTimezones(t *testing.T) {
	validZones := []string{"Asia/Shanghai", "UTC", "America/New_York"}

	for _, tz := range validZones {
		t.Run(tz, func(t *testing.T) {
			cfg := &StaticConfig{
				Port:            8080,
				LogLevel:        5,
				AppKey:          "test-key",
				DefaultTimezone: tz,
			}
			if err := cfg.Validate(); err != nil {
				t.Errorf("expected valid timezone %q, got error: %v", tz, err)
			}
		})
	}
}

func TestValidate_InvalidTimezone(t *testing.T) {
	cfg := &StaticConfig{
		Port:            8080,
		LogLevel:        5,
		AppKey:          "test-key",
		DefaultTimezone: "Invalid/Zone",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid timezone")
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

func TestLoadConfig_DefaultTimezone(t *testing.T) {
	// LoadConfig without a config file should apply the default timezone "Asia/Shanghai"
	// We need to provide a minimal valid app_key via environment variable
	t.Setenv("FLASHDUTY_MCP_APP_KEY", "test-key")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.DefaultTimezone != "Asia/Shanghai" {
		t.Errorf("expected default_timezone %q, got %q", "Asia/Shanghai", cfg.DefaultTimezone)
	}
}

func TestLoadConfig_InsecureSkipTLSVerify(t *testing.T) {
	t.Setenv("FLASHDUTY_MCP_APP_KEY", "test-key")
	t.Setenv("FLASHDUTY_MCP_INSECURE_SKIP_TLS_VERIFY", "true")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if !cfg.InsecureSkipTLSVerify {
		t.Error("expected insecure_skip_tls_verify to be true")
	}
}
