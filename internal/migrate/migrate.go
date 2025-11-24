package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"otp-core/internal/db"

	// register pgx driver for database/sql
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type Command string

const (
	CommandUp     Command = "up"
	CommandDown   Command = "down"
	CommandRedo   Command = "redo"
	CommandStatus Command = "status"
)

// Run executes goose migrations using the embedded SQL files.
func Run(ctx context.Context, cfg db.DatabaseConfig, command Command) error {
	dsn := cfg.DSN()
	log.Printf("migrate: start command=%s dsn=%s", command, dsn)

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open database for migrations: %w", err)
	}
	defer sqlDB.Close()

	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetMaxOpenConns(10)

	// Ensure migrations use embedded filesystem.
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	goose.SetTableName("schema_migrations")
	goose.SetLogger(log.New(log.Writer(), "goose: ", log.LstdFlags))
	goose.SetBaseFS(migrationsFS)

	var runErr error
	switch command {
	case CommandUp:
		runErr = goose.UpContext(ctx, sqlDB, migrationsDir)
	case CommandDown:
		runErr = goose.DownContext(ctx, sqlDB, migrationsDir)
	case CommandStatus:
		runErr = goose.StatusContext(ctx, sqlDB, migrationsDir)
	case CommandRedo:
		runErr = goose.RedoContext(ctx, sqlDB, migrationsDir)
	default:
		runErr = goose.RunContext(ctx, string(command), sqlDB, migrationsDir)
	}

	if runErr != nil {
		log.Printf("migrate: command=%s failed: %v", command, runErr)
		return runErr
	}

	log.Printf("migrate: command=%s finished successfully", command)
	return nil
}
