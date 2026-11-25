package db

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const connectTimeout = 10 * time.Second

// Postgres is a pgx pool that the DI container can shut down
// (it implements do.Shutdownable).
type Postgres struct {
	*pgxpool.Pool
}

// Shutdown closes the pool.
func (p *Postgres) Shutdown() error {
	p.Close()
	slog.Info("postgres: pool closed")
	return nil
}

// Redis is a redis client that the DI container can shut down
// (it implements do.Shutdownable).
type Redis struct {
	*redis.Client
}

// Shutdown closes the client.
func (r *Redis) Shutdown() error {
	slog.Info("redis: client closing")
	return r.Close()
}

// NewPostgres creates a connection pool and verifies connectivity.
func NewPostgres(ctx context.Context, cfg DatabaseConfig) (*Postgres, error) {
	dsnSafe := cfg.RedactedDSN()

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("postgres: parse dsn %s: %w", dsnSafe, err)
	}

	maxConns := runtime.NumCPU() * 4
	if cfg.MaxConns > 0 {
		maxConns = cfg.MaxConns
	}
	poolCfg.MaxConns = int32(min(maxConns, math.MaxInt32))

	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool for %s: %w", dsnSafe, err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping %s: %w", dsnSafe, err)
	}

	slog.Info("postgres: connected", "dsn", dsnSafe, "max_conns", poolCfg.MaxConns)
	return &Postgres{Pool: pool}, nil
}

// NewRedis creates a Redis client and verifies connectivity.
func NewRedis(ctx context.Context, cfg RedisConfig) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping %s: %w", cfg, err)
	}

	slog.Info("redis: connected", "addr", cfg.String())
	return &Redis{Client: client}, nil
}
