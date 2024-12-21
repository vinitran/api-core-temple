package db

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/redis/go-redis/v9"
)

// NewSQLDB creates a new SQL DB
func NewSQLDB(cfg DatabaseConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	max := runtime.NumCPU() * 4
	config.MaxConns = int32(max)

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	log.Printf("postgres: connecting to %s", dsn)

	return pool, err
}

// NewRedis creates a new REDIS DB
func NewRedis(cfg RedisConfig) (*redis.Client, error) {
	redisAddr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	log.Printf("redis: connecting to %s", redisAddr)

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Password,
		DB:       0,
	})

	err := rdb.Ping(context.Background()).Err()
	if err != nil {
		return nil, err
	}

	log.Printf("redis: connecting to %s", redisAddr)

	return rdb, nil
}
