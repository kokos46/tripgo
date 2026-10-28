package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	setValidEnvironment(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":8080" || cfg.DatabaseMaxConns != 10 || cfg.DatabaseMinConns != 2 {
		t.Fatalf("Load() returned unexpected config: %+v", cfg)
	}
	if cfg.DatabaseQueryTimeout != 3*time.Second || cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("Load() returned unexpected durations: %+v", cfg)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("DATABASE_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded without DATABASE_URL")
	}
}

func TestLoadRejectsPoolBounds(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("DATABASE_MIN_CONNS", "11")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded when minimum pool size exceeded maximum")
	}
}

func TestLoadUsesHTTPTimeoutDefaults(t *testing.T) {
	setValidEnvironment(t)
	for _, name := range []string{
		"HTTP_READ_TIMEOUT",
		"HTTP_READ_HEADER_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
	} {
		t.Setenv(name, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPReadTimeout != 10*time.Second ||
		cfg.HTTPReadHeaderTimeout != 5*time.Second ||
		cfg.HTTPWriteTimeout != 15*time.Second ||
		cfg.HTTPIdleTimeout != 60*time.Second {
		t.Fatalf("unexpected HTTP timeout defaults: %+v", cfg)
	}
}

func setValidEnvironment(t *testing.T) {
	t.Helper()
	values := map[string]string{
		"HTTP_ADDR":                  ":8080",
		"LOG_LEVEL":                  "info",
		"SHUTDOWN_TIMEOUT":           "10s",
		"HTTP_READ_TIMEOUT":          "10s",
		"HTTP_READ_HEADER_TIMEOUT":   "5s",
		"HTTP_WRITE_TIMEOUT":         "15s",
		"HTTP_IDLE_TIMEOUT":          "60s",
		"DATABASE_URL":               "postgres://user:pass@localhost:5432/tripgo",
		"DATABASE_MAX_CONNS":         "10",
		"DATABASE_MIN_CONNS":         "2",
		"DATABASE_MAX_CONN_LIFETIME": "30m",
		"DATABASE_CONNECT_TIMEOUT":   "5s",
		"DATABASE_QUERY_TIMEOUT":     "3s",
	}
	for name, value := range values {
		t.Setenv(name, value)
	}
}
