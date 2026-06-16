package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App struct {
		Name              string   `mapstructure:"name"`
		Env               string   `mapstructure:"env"`
		HTTPAddr          string   `mapstructure:"http_addr"`
		AdminUsername     string   `mapstructure:"admin_username"`
		AdminPassword     string   `mapstructure:"admin_password"`
		AdminPasswordHash string   `mapstructure:"admin_password_hash"`
		AllowedOrigins    []string `mapstructure:"allowed_origins"`
		EnableSwagger     bool     `mapstructure:"enable_swagger"`
	} `mapstructure:"app"`
	Bot struct {
		Token            string   `mapstructure:"token"`
		Mode             string   `mapstructure:"mode"`
		DefaultLocale    string   `mapstructure:"default_locale"`
		MiniAppURL       string   `mapstructure:"mini_app_url"`
		// DisabledFeatures lists module names to skip registering.
		// Valid values: verify, moderation, admin, points, lottery,
		//               publish, auto_reply, keywords, templates, invites.
		// Omit to enable everything (the default).
		DisabledFeatures []string `mapstructure:"disabled_features"`
	} `mapstructure:"bot"`
	Database DatabaseConfig `mapstructure:"database"`
	AiFilter  AiFilterConfig `mapstructure:"ai_filter"`
	Turnstile struct {
		SiteKey      string `mapstructure:"site_key"`
		SecretKey    string `mapstructure:"secret_key"`
		VerifySecret string `mapstructure:"verify_secret"`
	} `mapstructure:"turnstile"`
	Redis struct {
		Addr     string `mapstructure:"addr"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`
	JWT struct {
		Secret         string        `mapstructure:"secret"`
		Issuer         string        `mapstructure:"issuer"`
		AccessTokenTTL time.Duration `mapstructure:"access_token_ttl"`
	} `mapstructure:"jwt"`
}

type DatabaseConfig struct {
	DSN             string        `mapstructure:"dsn"`
	AutoMigrate     bool          `mapstructure:"auto_migrate"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}
// AiFilterConfig holds the AI-based spam/ad filter settings.
type AiFilterConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Provider string `mapstructure:"provider"`
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
	Endpoint string `mapstructure:"endpoint"`
}


var activeDatabaseConfig = DatabaseConfig{
	AutoMigrate:     false,
	MaxOpenConns:    20,
	MaxIdleConns:    5,
	ConnMaxLifetime: 30 * time.Minute,
}

func ActiveDatabaseConfig() DatabaseConfig {
	return activeDatabaseConfig
}

func Load(path string) (Config, error) {
	_ = godotenv.Load(".env")

	if path == "" {
		path = "config.yaml"
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetDefault("app.name", "sola")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.http_addr", ":8080")
	v.SetDefault("app.allowed_origins", []string{"http://127.0.0.1:5174", "http://localhost:5174"})
	v.SetDefault("bot.mode", "polling")
	v.SetDefault("bot.default_locale", "zh-CN")
	v.SetDefault("database.auto_migrate", false)
	v.SetDefault("database.max_open_conns", 20)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "30m")
	v.SetDefault("redis.addr", "redis:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("jwt.issuer", "sola-admin")
	v.SetDefault("jwt.access_token_ttl", "24h")

	v.SetEnvPrefix("SOLA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := bindEnv(v); err != nil {
		return Config{}, err
	}
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, statErr := os.Stat(path); statErr != nil {
			return Config{}, fmt.Errorf("load config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	origins := v.GetStringSlice("app.allowed_origins")
	rawOrigins := strings.TrimSpace(v.GetString("app.allowed_origins"))
	if strings.Contains(rawOrigins, ",") {
		origins = splitAndTrimCSV(rawOrigins)
	} else if len(origins) == 0 {
		origins = splitAndTrimCSV(rawOrigins)
	}
	if len(origins) > 0 {
		cfg.App.AllowedOrigins = origins
	}

	if cfg.JWT.AccessTokenTTL == 0 {
		ttl, err := time.ParseDuration("24h")
		if err != nil {
			return Config{}, err
		}
		cfg.JWT.AccessTokenTTL = ttl
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		lifetime, err := time.ParseDuration("30m")
		if err != nil {
			return Config{}, err
		}
		cfg.Database.ConnMaxLifetime = lifetime
	}
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 20
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 5
	}
	if err := validateProductionSecrets(cfg); err != nil {
		return Config{}, err
	}
	activeDatabaseConfig = cfg.Database

	return cfg, nil
}

func validateProductionSecrets(cfg Config) error {
	if !isProductionEnv(cfg.App.Env) {
		return nil
	}
	if strings.TrimSpace(cfg.JWT.Secret) == "" || strings.TrimSpace(cfg.JWT.Secret) == "change-this-in-production" {
		return fmt.Errorf("jwt.secret must be configured to a strong non-default value in production")
	}
	password := strings.TrimSpace(cfg.App.AdminPassword)
	if strings.TrimSpace(cfg.App.AdminPasswordHash) == "" && (password == "" || password == "change-me") {
		return fmt.Errorf("app.admin_password must not use the default value in production; configure app.admin_password_hash or a strong password")
	}
	return nil
}

func splitAndTrimCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func isProductionEnv(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "prod", "production":
		return true
	default:
		return false
	}
}

func bindEnv(v *viper.Viper) error {
	bindings := map[string]string{
		"app.name":                   "SOLA_APP_NAME",
		"app.env":                    "SOLA_APP_ENV",
		"app.http_addr":              "SOLA_APP_HTTP_ADDR",
		"app.admin_username":         "SOLA_APP_ADMIN_USERNAME",
		"app.admin_password":         "SOLA_APP_ADMIN_PASSWORD",
		"app.admin_password_hash":    "SOLA_APP_ADMIN_PASSWORD_HASH",
		"app.enable_swagger":         "SOLA_APP_ENABLE_SWAGGER",
		"bot.token":                  "SOLA_BOT_TOKEN",
		"bot.mode":                   "SOLA_BOT_MODE",
		"bot.default_locale":         "SOLA_BOT_DEFAULT_LOCALE",
		"bot.mini_app_url":           "SOLA_BOT_MINI_APP_URL",
		"database.dsn":               "SOLA_DATABASE_DSN",
		"database.auto_migrate":      "SOLA_DATABASE_AUTO_MIGRATE",
		"database.max_open_conns":    "SOLA_DATABASE_MAX_OPEN_CONNS",
		"database.max_idle_conns":    "SOLA_DATABASE_MAX_IDLE_CONNS",
		"database.conn_max_lifetime": "SOLA_DATABASE_CONN_MAX_LIFETIME",
		"redis.addr":                 "SOLA_REDIS_ADDR",
		"redis.password":             "SOLA_REDIS_PASSWORD",
		"redis.db":                   "SOLA_REDIS_DB",
		"jwt.secret":                 "SOLA_JWT_SECRET",
		"jwt.issuer":                 "SOLA_JWT_ISSUER",
		"ai_filter.enabled":            "SOLA_AI_FILTER_ENABLED",
		"ai_filter.provider":           "SOLA_AI_FILTER_PROVIDER",
		"ai_filter.api_key":            "SOLA_AI_FILTER_API_KEY",
		"ai_filter.model":              "SOLA_AI_FILTER_MODEL",
		"ai_filter.endpoint":           "SOLA_AI_FILTER_ENDPOINT",
		"turnstile.site_key":           "SOLA_TURNSTILE_SITE_KEY",
		"turnstile.secret_key":         "SOLA_TURNSTILE_SECRET_KEY",
		"turnstile.verify_secret":      "SOLA_TURNSTILE_VERIFY_SECRET",
		"jwt.access_token_ttl":         "SOLA_JWT_ACCESS_TOKEN_TTL",
	}

	for key, env := range bindings {
		if err := v.BindEnv(key, env); err != nil {
			return fmt.Errorf("bind env %s: %w", env, err)
		}
	}
	if err := v.BindEnv("app.allowed_origins", "SOLA_APP_ALLOWED_ORIGINS"); err != nil {
		return fmt.Errorf("bind env %s: %w", "SOLA_APP_ALLOWED_ORIGINS", err)
	}
	return nil
}
