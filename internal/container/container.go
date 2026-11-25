// Package container wires the application's dependencies. It is the only
// place that knows concrete implementations; every other package depends on
// interfaces it owns.
package container

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/samber/do"

	"api-core/internal/config"
	"api-core/internal/datastore"
	"api-core/internal/datastore/oauthstate"
	"api-core/internal/datastore/userstore"
	"api-core/internal/db"
	"api-core/internal/handler"
	authhandler "api-core/internal/handler/auth"
	authservice "api-core/internal/service/auth"
	appauth "api-core/pkg/auth"
	"api-core/pkg/jwtx"
)

// New registers all providers. Providers are lazy: nothing connects until a
// service is invoked, so commands only pay for what they use. Call
// injector.Shutdown() to close the connections that were opened.
func New(cfg *config.Config) *do.Injector {
	i := do.New()
	do.ProvideValue(i, cfg)

	provideInfrastructure(i)
	provideAuth(i)
	provideHTTP(i)

	return i
}

func provideInfrastructure(i *do.Injector) {
	do.Provide(i, func(i *do.Injector) (*db.Postgres, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return db.NewPostgres(context.Background(), cfg.Database)
	})

	do.Provide(i, func(i *do.Injector) (*db.Redis, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return db.NewRedis(context.Background(), cfg.Redis)
	})

	do.Provide(i, func(i *do.Injector) (datastore.TxRunner, error) {
		pg, err := do.Invoke[*db.Postgres](i)
		if err != nil {
			return nil, err
		}
		return datastore.NewTxRunner(pg), nil
	})
}

func provideAuth(i *do.Injector) {
	do.Provide(i, func(i *do.Injector) (*userstore.Store, error) {
		pg, err := do.Invoke[*db.Postgres](i)
		if err != nil {
			return nil, err
		}
		return userstore.New(pg), nil
	})

	do.Provide(i, func(i *do.Injector) (*oauthstate.RedisStore, error) {
		rdb, err := do.Invoke[*db.Redis](i)
		if err != nil {
			return nil, err
		}
		return oauthstate.NewRedisStore(rdb.Client), nil
	})

	do.Provide(i, func(i *do.Injector) (*jwtx.HMAC, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return jwtx.NewHMAC(jwtx.HMACConfig{
			Secret:   cfg.Auth.JWTSecret,
			Issuer:   cfg.Auth.JWTIssuer,
			Audience: cfg.Auth.JWTAudience,
			TTL:      cfg.Auth.JWTExpiration,
		})
	})

	do.Provide(i, func(i *do.Injector) (*authservice.Service, error) {
		cfg := do.MustInvoke[*config.Config](i)

		users, err := do.Invoke[*userstore.Store](i)
		if err != nil {
			return nil, err
		}
		states, err := do.Invoke[*oauthstate.RedisStore](i)
		if err != nil {
			return nil, err
		}
		tokens, err := do.Invoke[*jwtx.HMAC](i)
		if err != nil {
			return nil, err
		}
		identity, err := newGoogleProvider(cfg.Google)
		if err != nil {
			return nil, err
		}

		return authservice.NewService(authservice.Deps{
			Users:    users,
			Identity: identity,
			States:   states,
			Tokens:   tokens,
		})
	})

	do.Provide(i, func(i *do.Injector) (*authhandler.Handler, error) {
		svc, err := do.Invoke[*authservice.Service](i)
		if err != nil {
			return nil, err
		}
		return authhandler.NewHandler(svc), nil
	})
}

// newGoogleProvider returns a nil interface (not a typed nil) when Google
// sign-in is not configured, so the service can detect it.
func newGoogleProvider(cfg config.GoogleConfig) (authservice.IdentityProvider, error) {
	if !cfg.Enabled() {
		slog.Warn("google oauth is not configured; /auth/google endpoints will return 503")
		return nil, nil
	}

	client, err := appauth.NewGoogleOAuth(appauth.GoogleOAuthConfig{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
	})
	if err != nil {
		return nil, err
	}

	return authservice.NewGoogleProvider(client), nil
}

func provideHTTP(i *do.Injector) {
	do.Provide(i, func(i *do.Injector) (http.Handler, error) {
		cfg := do.MustInvoke[*config.Config](i)

		verifier, err := do.Invoke[*jwtx.HMAC](i)
		if err != nil {
			return nil, err
		}
		auth, err := do.Invoke[*authhandler.Handler](i)
		if err != nil {
			return nil, err
		}

		return handler.New(handler.Config{
			AllowedOrigins: cfg.HTTP.AllowedOrigins,
			Verifier:       verifier,
			Modules:        []handler.Module{auth},
		})
	})
}
