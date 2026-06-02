package platform

import (
	"fmt"
	"time"

	"github.com/dabowin/sola/internal/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func OpenPostgres(dsn string, poolConfigs ...PostgresPoolConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := applyPostgresPoolConfig(db, selectedPostgresPoolConfig(poolConfigs...)); err != nil {
		return nil, err
	}
	return db, nil
}

func selectedPostgresPoolConfig(poolConfigs ...PostgresPoolConfig) PostgresPoolConfig {
	if len(poolConfigs) > 0 {
		return normalizePostgresPoolConfig(poolConfigs[0])
	}
	database := config.ActiveDatabaseConfig()
	return normalizePostgresPoolConfig(PostgresPoolConfig{
		MaxOpenConns:    database.MaxOpenConns,
		MaxIdleConns:    database.MaxIdleConns,
		ConnMaxLifetime: database.ConnMaxLifetime,
	})
}

func normalizePostgresPoolConfig(pool PostgresPoolConfig) PostgresPoolConfig {
	if pool.MaxOpenConns == 0 {
		pool.MaxOpenConns = 20
	}
	if pool.MaxIdleConns == 0 {
		pool.MaxIdleConns = 5
	}
	if pool.ConnMaxLifetime == 0 {
		pool.ConnMaxLifetime = 30 * time.Minute
	}
	return pool
}

func applyPostgresPoolConfig(db *gorm.DB, pool PostgresPoolConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("open postgres sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(pool.MaxOpenConns)
	sqlDB.SetMaxIdleConns(pool.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(pool.ConnMaxLifetime)
	return nil
}

func OpenRedis(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}
