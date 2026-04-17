package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaultsWhenFileIsMissing(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.App.Name != "go-seckill" {
		t.Fatalf("expected default app name, got %q", cfg.App.Name)
	}

	if cfg.Server.Port != 8080 {
		t.Fatalf("expected default port 8080, got %d", cfg.Server.Port)
	}
}

func TestLoadAllowsEnvOverride(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`app:
  name: from-file
server:
  port: 8088
log:
  level: warn
mysql:
  host: mysql-from-file
redis:
  addr: redis-from-file:6379
`)

	if err := os.WriteFile(configFile, content, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("GO_SECKILL_APP_NAME", "from-env")
	t.Setenv("GO_SECKILL_SERVER_PORT", "9090")

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.App.Name != "from-env" {
		t.Fatalf("expected env to override app name, got %q", cfg.App.Name)
	}

	if cfg.Server.Port != 9090 {
		t.Fatalf("expected env to override port, got %d", cfg.Server.Port)
	}

	if cfg.Log.Level != "warn" {
		t.Fatalf("expected log level to come from file, got %q", cfg.Log.Level)
	}

	if cfg.MySQL.Host != "mysql-from-file" {
		t.Fatalf("expected mysql host to come from file, got %q", cfg.MySQL.Host)
	}

	if cfg.Redis.Addr != "redis-from-file:6379" {
		t.Fatalf("expected redis addr to come from file, got %q", cfg.Redis.Addr)
	}
}
