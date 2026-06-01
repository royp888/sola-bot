package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadReadsLocalDotEnvWithSOLAPrefix(t *testing.T) {
	tmp := t.TempDir()
	restoreWorkingDir := chdir(t, tmp)
	defer restoreWorkingDir()
	restoreEnv := clearEnv(t,
		"SOLA_APP_HTTP_ADDR",
		"SOLA_BOT_TOKEN",
		"SOLA_BOT_MODE",
		"SOLA_DATABASE_DSN",
		"SOLA_DATABASE_AUTO_MIGRATE",
		"SOLA_REDIS_ADDR",
		"SOLA_REDIS_DB",
		"SOLA_JWT_SECRET",
		"SOLA_JWT_ACCESS_TOKEN_TTL",
	)
	defer restoreEnv()

	writeFile(t, ".env", `
SOLA_APP_HTTP_ADDR=:9090
SOLA_BOT_TOKEN=placeholder-token-from-env
SOLA_BOT_MODE=webhook
SOLA_DATABASE_DSN=postgres://env-user:env-pass@localhost:5432/envdb?sslmode=disable
SOLA_DATABASE_AUTO_MIGRATE=false
SOLA_REDIS_ADDR=localhost:6380
SOLA_REDIS_DB=3
SOLA_JWT_SECRET=placeholder-jwt-secret
SOLA_JWT_ACCESS_TOKEN_TTL=2h
`)
	writeFile(t, "config.yaml", `
app:
  http_addr: :8080
bot:
  token: placeholder-token-from-yaml
  mode: polling
database:
  dsn: postgres://yaml-user:yaml-pass@postgres:5432/yamldb?sslmode=disable
  auto_migrate: true
redis:
  addr: redis:6379
  db: 0
jwt:
  secret: placeholder-yaml-secret
  access_token_ttl: 24h
`)

	cfg, err := Load(filepath.Join(tmp, "config.yaml"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.App.HTTPAddr != ":9090" {
		t.Fatalf("App.HTTPAddr = %q, want :9090", cfg.App.HTTPAddr)
	}
	if cfg.Bot.Token != "placeholder-token-from-env" {
		t.Fatalf("Bot.Token = %q, want env token", cfg.Bot.Token)
	}
	if cfg.Bot.Mode != "webhook" {
		t.Fatalf("Bot.Mode = %q, want webhook", cfg.Bot.Mode)
	}
	if cfg.Database.DSN != "postgres://env-user:env-pass@localhost:5432/envdb?sslmode=disable" {
		t.Fatalf("Database.DSN = %q, want env DSN", cfg.Database.DSN)
	}
	if cfg.Database.AutoMigrate {
		t.Fatalf("Database.AutoMigrate = true, want false")
	}
	if cfg.Redis.Addr != "localhost:6380" {
		t.Fatalf("Redis.Addr = %q, want localhost:6380", cfg.Redis.Addr)
	}
	if cfg.Redis.DB != 3 {
		t.Fatalf("Redis.DB = %d, want 3", cfg.Redis.DB)
	}
	if cfg.JWT.Secret != "placeholder-jwt-secret" {
		t.Fatalf("JWT.Secret = %q, want env secret", cfg.JWT.Secret)
	}
	if cfg.JWT.AccessTokenTTL != 2*time.Hour {
		t.Fatalf("JWT.AccessTokenTTL = %s, want 2h", cfg.JWT.AccessTokenTTL)
	}
}

func chdir(t *testing.T, dir string) func() {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	return func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}
}

func clearEnv(t *testing.T, keys ...string) func() {
	t.Helper()
	previous := make(map[string]string, len(keys))
	present := make(map[string]bool, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			previous[key] = value
			present[key] = true
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
	return func() {
		for _, key := range keys {
			var err error
			if present[key] {
				err = os.Setenv(key, previous[key])
			} else {
				err = os.Unsetenv(key)
			}
			if err != nil {
				t.Fatalf("restore %s: %v", key, err)
			}
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
