package container

import (
	"net/http"
	"otp-core/internal/config"
	"otp-core/internal/content/handler"
	"otp-core/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
)

func NewContainer(cfg *config.Config) (*do.Injector, error) {
	injector := do.New()

	do.ProvideValue(injector, cfg)

	do.Provide(injector, func(i *do.Injector) (*pgxpool.Pool, error) {
		return db.NewSQLDB(cfg.Database)
	})
	do.Provide(injector, func(i *do.Injector) (*redis.Client, error) {
		return db.NewRedis(cfg.Redis)
	})

	do.Provide(injector, ProvideRouter)

	if _, err := do.Invoke[*pgxpool.Pool](injector); err != nil {
		return nil, err
	}

	if _, err := do.Invoke[*redis.Client](injector); err != nil {
		return nil, err
	}

	return injector, nil
}

func ProvideRouter(i *do.Injector) (http.Handler, error) {
	return handler.New(&handler.Config{
		Container: i,
		Origins:   []string{"*"},
	})
}
