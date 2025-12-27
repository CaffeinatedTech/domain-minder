package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	os.Clearenv()

	os.Setenv("DB_PATH", "/test/db.sqlite")
	os.Setenv("PORT", "9999")
	os.Setenv("SESSION_SECRET", "test-secret")
	os.Setenv("CHECK_INTERVAL", "1h")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DBPath != "/test/db.sqlite" {
		t.Errorf("DBPath = %v, want /test/db.sqlite", cfg.DBPath)
	}

	if cfg.Port != 9999 {
		t.Errorf("Port = %v, want 9999", cfg.Port)
	}

	if cfg.SessionSecret != "test-secret" {
		t.Errorf("SessionSecret = %v, want test-secret", cfg.SessionSecret)
	}

	if cfg.CheckInterval.Hours() != 1 {
		t.Errorf("CheckInterval = %v, want 1h", cfg.CheckInterval)
	}

	os.Unsetenv("DB_PATH")
	os.Unsetenv("PORT")
	os.Unsetenv("SESSION_SECRET")
	os.Unsetenv("CHECK_INTERVAL")
}

func TestLoadDefaults(t *testing.T) {
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DBPath != "data/domain_minder.db" {
		t.Errorf("Default DBPath = %v, want data/domain_minder.db", cfg.DBPath)
	}

	if cfg.Port != 9000 {
		t.Errorf("Default Port = %v, want 9000", cfg.Port)
	}
}

func TestServerAddr(t *testing.T) {
	cfg := &Config{Port: 8080}
	expected := ":8080"
	if cfg.ServerAddr() != expected {
		t.Errorf("ServerAddr() = %v, want %v", cfg.ServerAddr(), expected)
	}
}

func TestCheckIntervalString(t *testing.T) {
	tests := []struct {
		name     string
		interval interface{}
		expected string
	}{
		{
			name:     "zero interval",
			interval: 0,
			expected: "0 0 */6 * * *",
		},
		{
			name:     "6 hour interval",
			interval: 6 * 3600 * 1000000000,
			expected: "0 0 */6 * * *",
		},
		{
			name:     "24 hour interval",
			interval: 24 * 3600 * 1000000000,
			expected: "0 0 */24 * * *",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			if v, ok := tt.interval.(int); ok {
				cfg.CheckInterval = time.Duration(v)
			}
			result := cfg.CheckIntervalString()
			if result != tt.expected {
				t.Errorf("CheckIntervalString() = %v, want %v", result, tt.expected)
			}
		})
	}
}
