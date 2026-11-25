// Package migrate applies the embedded SQL migrations with goose.
package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	// register pgx driver for database/sql
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"api-core/internal/db"
)

const tableName = "schema_migrations"

type Command string

const (
	CommandUp      Command = "up"
	CommandDown    Command = "down"
	CommandRedo    Command = "redo"
	CommandStatus  Command = "status"
	CommandVersion Command = "version"
)

// Run executes a migration command against the embedded SQL files.
//
// A Postgres advisory lock is held for the duration of the command, so several
// API instances starting at the same time apply migrations exactly once.
func Run(ctx context.Context, cfg db.DatabaseConfig, command Command) (err error) {
	logger := slog.With("component", "migrate", "command", command)
	logger.Info("migrate: start", "dsn", cfg.RedactedDSN())

	sqlDB, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return fmt.Errorf("migrate: open database: %w", err)
	}
	defer func() {
		err = errors.Join(err, sqlDB.Close())
	}()
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetMaxOpenConns(4)

	provider, err := newProvider(sqlDB)
	if err != nil {
		return err
	}

	if err := run(ctx, provider, command, logger); err != nil {
		logger.Error("migrate: failed", "error", err)
		return fmt.Errorf("migrate %s: %w", command, err)
	}

	logger.Info("migrate: finished")
	return nil
}

func newProvider(sqlDB *sql.DB) (*goose.Provider, error) {
	migrations, err := fs.Sub(migrationsFS, migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("migrate: open embedded migrations: %w", err)
	}

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("migrate: create session locker: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations,
		goose.WithTableName(tableName),
		goose.WithSessionLocker(locker),
		goose.WithSlog(slog.Default()),
	)
	if err != nil {
		return nil, fmt.Errorf("migrate: create provider: %w", err)
	}

	return provider, nil
}

func run(ctx context.Context, p *goose.Provider, command Command, logger *slog.Logger) error {
	switch command {
	case CommandUp:
		results, err := p.Up(ctx)
		if err != nil {
			return err
		}
		logger.Info("migrate: applied", "count", len(results))
		return nil

	case CommandDown:
		_, err := p.Down(ctx)
		return err

	case CommandRedo:
		if _, err := p.Down(ctx); err != nil {
			return err
		}
		_, err := p.UpByOne(ctx)
		return err

	case CommandStatus:
		statuses, err := p.Status(ctx)
		if err != nil {
			return err
		}
		for _, s := range statuses {
			logger.Info("migration", "version", s.Source.Version, "file", s.Source.Path, "state", s.State, "applied_at", s.AppliedAt)
		}
		return nil

	case CommandVersion:
		version, err := p.GetDBVersion(ctx)
		if err != nil {
			return err
		}
		logger.Info("migrate: current version", "version", version)
		return nil
	}

	return fmt.Errorf("unknown command %q (want up, down, redo, status or version)", command)
}
