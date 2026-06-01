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
	} `mapstructure:"app"`
	Bot struct {
		Token         string `mapstructure:"token"`
		Mode          string `mapstructure:"mode"`
		DefaultLocale string `mapstructure:"default_locale"`
	} `mapstructure:"bot"`
	Database struct {
		DSN         string `mapstructure:"dsn"`
		AutoMigrate bool   `mapstructure:"auto_migrate"`
	} `mapstructure:"database"`
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
	v.SetDefault("database.auto_migrate", true)
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

	if cfg.JWT.AccessTokenTTL == 0 {
		ttl, err := time.ParseDuration("24h")
		if err != nil {
			return Config{}, err
		}
		cfg.JWT.AccessTokenTTL = ttl
	}

	return cfg, nil
}

func bindEnv(v *viper.Viper) error {
	bindings := map[string]string{
		"app.name":                "SOLA_APP_NAME",
		"app.env":                 "SOLA_APP_ENV",
		"app.http_addr":           "SOLA_APP_HTTP_ADDR",
		"app.admin_username":      "SOLA_APP_ADMIN_USERNAME",
		"app.admin_password":      "SOLA_APP_ADMIN_PASSWORD",
		"app.admin_password_hash": "SOLA_APP_ADMIN_PASSWORD_HASH",
		"bot.token":               "SOLA_BOT_TOKEN",
		"bot.mode":                "SOLA_BOT_MODE",
		"bot.default_locale":      "SOLA_BOT_DEFAULT_LOCALE",
		"database.dsn":            "SOLA_DATABASE_DSN",
		"database.auto_migrate":   "SOLA_DATABASE_AUTO_MIGRATE",
		"redis.addr":              "SOLA_REDIS_ADDR",
		"redis.password":          "SOLA_REDIS_PASSWORD",
		"redis.db":                "SOLA_REDIS_DB",
		"jwt.secret":              "SOLA_JWT_SECRET",
		"jwt.issuer":              "SOLA_JWT_ISSUER",
		"jwt.access_token_ttl":    "SOLA_JWT_ACCESS_TOKEN_TTL",
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
