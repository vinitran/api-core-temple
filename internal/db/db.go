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
	dsnSafe := fmt.Sprintf("postgres://%s:****@%s:%s/%s?sslmode=disable", cfg.User, cfg.Host, cfg.Port, cfg.Name)

	log.Printf("postgres: initializing pool for %s", dsnSafe)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Printf("postgres: failed to parse DSN %s: %v", dsnSafe, err)
		return nil, err
	}

	max := runtime.NumCPU() * 4
	if cfg.MaxConns > 0 {
		max = cfg.MaxConns
	}
	config.MaxConns = int32(max)

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Printf("postgres: failed to create pool for %s: %v", dsnSafe, err)
		return nil, err
	}

	err = pool.Ping(context.Background())
	if err != nil {
		log.Printf("postgres: ping failed for %s: %v", dsnSafe, err)
		return nil, err
	}

	log.Printf("postgres: connected successfully to %s", dsnSafe)
	return pool, err
}

// NewRedis creates a new REDIS DB
func NewRedis(cfg RedisConfig) (*redis.Client, error) {
	redisAddr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	log.Printf("redis: initializing client for %s", redisAddr)

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Password,
		DB:       0,
	})

	err := rdb.Ping(context.Background()).Err()
	if err != nil {
		log.Printf("redis: ping failed for %s: %v", redisAddr, err)
		return nil, err
	}

	log.Printf("redis: connected successfully to %s", redisAddr)
	return rdb, nil
}
