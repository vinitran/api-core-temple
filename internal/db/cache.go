package db

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"
)

// Cache is a typed-agnostic key/value cache.
type Cache interface {
	Get(ctx context.Context, key string, target any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// UseCache returns the cached value for key, or computes it with callback and
// stores it. A cache outage degrades to calling callback instead of failing.
func UseCache[T any](ctx context.Context, c Cache, key string, ttl time.Duration, callback func() (T, error)) (T, error) {
	var v T
	err := c.Get(ctx, key, &v)
	if err == nil {
		return v, nil
	}
	if !errors.Is(err, cache.ErrCacheMiss) {
		slog.WarnContext(ctx, "cache get failed, falling back to source", "key", key, "error", err)
	}

	v, err = callback()
	if err != nil {
		return v, err
	}

	if err := c.Set(ctx, key, v, ttl); err != nil {
		slog.WarnContext(ctx, "cache set failed", "key", key, "error", err)
	}

	return v, nil
}

// GetCache returns the cached value, or the zero value on a miss.
func GetCache[T any](ctx context.Context, c Cache, key string) (T, error) {
	var v T
	err := c.Get(ctx, key, &v)
	if err != nil && !errors.Is(err, cache.ErrCacheMiss) {
		return v, err
	}

	return v, nil
}

// CacheRedis is a two-tier cache: an in-process TinyLFU in front of Redis.
type CacheRedis struct {
	instance *cache.Cache
}

var _ Cache = (*CacheRedis)(nil)

func NewCacheRedis(client *redis.Client) *CacheRedis {
	return &CacheRedis{cache.New(&cache.Options{
		Redis:      client,
		LocalCache: cache.NewTinyLFU(10000, time.Minute),
	})}
}

func (c *CacheRedis) Get(ctx context.Context, key string, target any) error {
	return c.instance.Get(ctx, key, target)
}

func (c *CacheRedis) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return c.instance.Set(&cache.Item{
		Ctx:   ctx,
		Key:   key,
		Value: value,
		TTL:   ttl,
	})
}

func (c *CacheRedis) Delete(ctx context.Context, key string) error {
	return c.instance.Delete(ctx, key)
}
