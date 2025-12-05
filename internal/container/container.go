package container

import (
	"api-core/internal/config"
	"api-core/internal/datastore"
	"api-core/internal/datastore/userstore"
	"api-core/internal/db"
	"api-core/internal/handler"
	authhandler "api-core/internal/handler/auth"
	authservice "api-core/internal/service/auth"
	appauth "api-core/pkg/auth"
	"api-core/pkg/jwtx"
	"net/http"

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

	do.Provide(injector, func(i *do.Injector) (datastore.TxRunner, error) {
		pool, err := do.Invoke[*pgxpool.Pool](i)
		if err != nil {
			return nil, err
		}
		return datastore.NewTxRunner(pool), nil
	})

	do.Provide(injector, func(i *do.Injector) (userstore.Store, error) {
		pool, err := do.Invoke[*pgxpool.Pool](i)
		if err != nil {
			return nil, err
		}
		return userstore.New(pool), nil
	})

	do.Provide(injector, func(i *do.Injector) (*appauth.GoogleOAuth, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return appauth.NewGoogleOAuth(appauth.GoogleOAuthConfig{
			ClientID:     cfg.Google.ClientID,
			ClientSecret: cfg.Google.ClientSecret,
			RedirectURL:  cfg.Google.RedirectURL,
			Scopes:       cfg.Google.Scopes,
		})
	})

	do.Provide(injector, func(i *do.Injector) (*jwtx.HMACIssuer, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return jwtx.NewHMACIssuer(cfg.Auth.JWTSecret, cfg.Auth.JWTIssuer, cfg.Auth.JWTExpiration)
	})

	do.Provide(injector, func(i *do.Injector) (*authservice.Service, error) {
		repo := do.MustInvoke[userstore.Store](i)
		googleOAuth := do.MustInvoke[*appauth.GoogleOAuth](i)
		tokenIssuer := do.MustInvoke[*jwtx.HMACIssuer](i)
		redisClient := do.MustInvoke[*redis.Client](i)
		cfg := do.MustInvoke[*config.Config](i)
		return authservice.NewService(repo, googleOAuth, tokenIssuer, redisClient, cfg.Google), nil
	})

	do.Provide(injector, func(i *do.Injector) (*authhandler.Handler, error) {
		service := do.MustInvoke[*authservice.Service](i)
		return authhandler.NewHandler(service), nil
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
