package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samber/do"
	"github.com/urfave/cli/v2"

	"api-core/internal/config"
	"api-core/internal/container"
	"api-core/internal/migrate"
)

const (
	shutdownTimeout   = 15 * time.Second
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
)

func apiCommand() *cli.Command {
	return &cli.Command{
		Name:  "api",
		Usage: "Run the HTTP API",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  config.FlagAddress,
				Value: "0.0.0.0:3030",
				Usage: "listen address",
			},
			&cli.BoolFlag{
				Name:    config.FlagSkipMigrate,
				Usage:   "do not apply pending migrations before starting",
				EnvVars: []string{"SKIP_MIGRATE"},
			},
		},
		Action: runAPI,
	}
}

func runAPI(c *cli.Context) (err error) {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Migrations always run before serving, so the schema matches the code.
	// Safe with multiple replicas: goose holds a Postgres advisory lock.
	if c.Bool(config.FlagSkipMigrate) {
		slog.Warn("skipping migrations", "flag", config.FlagSkipMigrate)
	} else if err := migrate.Run(c.Context, cfg.Database, migrate.CommandUp); err != nil {
		return err
	}

	injector := container.New(cfg)
	defer func() {
		if shutdownErr := injector.Shutdown(); shutdownErr != nil {
			slog.Error("shutdown dependencies", "error", shutdownErr)
		}
	}()

	router, err := do.Invoke[http.Handler](injector)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              c.String(config.FlagAddress),
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	ctx, stop := signal.NotifyContext(c.Context, os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down http server", "timeout", shutdownTimeout)
	// The shutdown deadline starts now, not at process start.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return srv.Shutdown(shutdownCtx)
}
