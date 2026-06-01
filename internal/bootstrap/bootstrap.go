package bootstrap

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/config"
	"github.com/dabowin/sola/internal/platform"
	"github.com/dabowin/sola/internal/service"
	"github.com/dabowin/sola/internal/store"
)

type Resources struct {
	Config   config.Config
	Logger   *zap.Logger
	DB       interface{ Close() error }
	Store    *store.Store
	Redis    *redis.Client
	Services *service.Bundle
}

func New(ctx context.Context, path string) (*Resources, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}

	log, err := platform.NewLogger(cfg.App.Env)
	if err != nil {
		return nil, err
	}

	db, err := platform.OpenPostgres(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	rdb := platform.OpenRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Warn("redis ping failed", zap.Error(err))
		_ = rdb.Close()
		rdb = nil
	}

	st := store.New(db, rdb)
	if cfg.Database.AutoMigrate {
		if err := st.AutoMigrate(ctx); err != nil {
			return nil, fmt.Errorf("auto migrate: %w", err)
		}
	}

	bundle := service.NewBundleWithBotToken(st, rdb, cfg.Bot.Token)

	return &Resources{
		Config:   cfg,
		Logger:   log,
		DB:       sqlDB,
		Store:    st,
		Redis:    rdb,
		Services: bundle,
	}, nil
}

func (r *Resources) Close(ctx context.Context) {
	if r == nil {
		return
	}
	if r.Redis != nil {
		_ = r.Redis.Close()
	}
	if r.Logger != nil {
		_ = r.Logger.Sync()
	}
}

func (r *Resources) BotServices() bot.Services {
	if r == nil || r.Services == nil {
		return bot.Services{}
	}
	return r.Services.BotServices()
}
