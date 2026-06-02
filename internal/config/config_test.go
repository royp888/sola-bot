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
		"SOLA_APP_ENV",
		"SOLA_APP_HTTP_ADDR",
		"SOLA_APP_ADMIN_PASSWORD",
		"SOLA_APP_ADMIN_PASSWORD_HASH",
		"SOLA_APP_ENABLE_SWAGGER",
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
SOLA_APP_ENABLE_SWAGGER=true
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
	if !cfg.App.EnableSwagger {
		t.Fatalf("App.EnableSwagger = false, want true")
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

func TestLoadRejectsProductionDefaultSecrets(t *testing.T) {
	tmp := t.TempDir()
	restoreWorkingDir := chdir(t, tmp)
	defer restoreWorkingDir()
	restoreEnv := clearEnv(t,
		"SOLA_APP_ENV",
		"SOLA_APP_ADMIN_PASSWORD",
		"SOLA_APP_ADMIN_PASSWORD_HASH",
		"SOLA_JWT_SECRET",
	)
	defer restoreEnv()

	writeFile(t, "config.yaml", `
app:
  env: production
  admin_password: change-me
jwt:
  secret: change-this-in-production
`)

	if _, err := Load(filepath.Join(tmp, "config.yaml")); err == nil {
		t.Fatal("Load returned nil error, want production default secret rejection")
	}
}

func TestLoadAllowsProductionStrongSecrets(t *testing.T) {
	tmp := t.TempDir()
	restoreWorkingDir := chdir(t, tmp)
	defer restoreWorkingDir()
	restoreEnv := clearEnv(t,
		"SOLA_APP_ENV",
		"SOLA_APP_ADMIN_PASSWORD",
		"SOLA_APP_ADMIN_PASSWORD_HASH",
		"SOLA_JWT_SECRET",
	)
	defer restoreEnv()

	writeFile(t, "config.yaml", `
app:
  env: production
  admin_password_hash: "$2y$12$JYBAhS0Hj4pQcjT6qcI/R.RzcT1vzP/MQkI.MJCKnJxYeUHiN4B/i"
jwt:
  secret: production-secret-with-enough-randomness
`)

	if _, err := Load(filepath.Join(tmp, "config.yaml")); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
}

func TestLoadParsesAllowedOriginsCSVFromEnv(t *testing.T) {
	tmp := t.TempDir()
	restoreWorkingDir := chdir(t, tmp)
	defer restoreWorkingDir()
	restoreEnv := clearEnv(t,
		"SOLA_APP_ALLOWED_ORIGINS",
	)
	defer restoreEnv()

	writeFile(t, "config.yaml", `
app:
  allowed_origins:
    - http://localhost:3000
`)
	if err := os.Setenv("SOLA_APP_ALLOWED_ORIGINS", " http://127.0.0.1:5174,  , http://localhost:5174 "); err != nil {
		t.Fatalf("Setenv returned error: %v", err)
	}

	cfg, err := Load(filepath.Join(tmp, "config.yaml"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(cfg.App.AllowedOrigins) != 2 {
		t.Fatalf("len(App.AllowedOrigins) = %d, want 2: %+v", len(cfg.App.AllowedOrigins), cfg.App.AllowedOrigins)
	}
	if cfg.App.AllowedOrigins[0] != "http://127.0.0.1:5174" || cfg.App.AllowedOrigins[1] != "http://localhost:5174" {
		t.Fatalf("App.AllowedOrigins = %+v, want parsed CSV origins", cfg.App.AllowedOrigins)
	}
}

func TestLoadRejectsProductionEmptyAdminPasswordWithoutHash(t *testing.T) {
	tmp := t.TempDir()
	restoreWorkingDir := chdir(t, tmp)
	defer restoreWorkingDir()
	restoreEnv := clearEnv(t,
		"SOLA_APP_ENV",
		"SOLA_APP_ADMIN_PASSWORD",
		"SOLA_APP_ADMIN_PASSWORD_HASH",
		"SOLA_JWT_SECRET",
	)
	defer restoreEnv()

	writeFile(t, "config.yaml", `
app:
  env: production
  admin_password: ""
jwt:
  secret: production-secret-with-enough-randomness
`)

	if _, err := Load(filepath.Join(tmp, "config.yaml")); err == nil {
		t.Fatal("Load returned nil error, want production empty admin password rejection")
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
