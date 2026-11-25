package main

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

const (
	appName = "api-core"
	envPath = ".env"
)

func main() {
	setupLogger(os.Getenv("LOG_FORMAT"), os.Getenv("LOG_LEVEL"))

	// .env is optional: in containers configuration comes from the environment.
	if err := godotenv.Load(envPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Error("load .env", "error", err)
		os.Exit(1)
	}

	app := &cli.App{
		Name:     appName,
		Commands: []*cli.Command{apiCommand(), migrateCommand()},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func setupLogger(format, level string) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var h slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	if strings.EqualFold(format, "json") {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(h))
}
