package main

import (
	"github.com/urfave/cli/v2"

	"api-core/internal/config"
	"api-core/internal/migrate"
)

func migrateCommand() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Run database migrations (goose)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  config.FlagMigrateAction,
				Value: config.FlagUpAction,
				Usage: "migration action (up, down, redo, status, ...)",
			},
		},
		Action: func(c *cli.Context) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Only Postgres is needed here; no DI container, no Redis.
			return migrate.Run(c.Context, cfg.Database, migrate.Command(c.String(config.FlagMigrateAction)))
		},
	}
}
