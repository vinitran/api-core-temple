package main

import (
	"log"
	"os"
	"otp-core/internal/config"
	"otp-core/internal/container"
	"otp-core/internal/migrate"
	"strings"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

const (
	appName = "backend-wallet"
	envPath = ".env"
)

var (
	configAddressFlag = cli.StringFlag{
		Name:     config.FlagAddress,
		Value:    "0.0.0.0:3030",
		Usage:    "Configuration Address",
		Required: false,
	}
	configMigrateActionFlag = cli.StringFlag{
		Name:     config.FlagMigrateAction,
		Value:    config.FlagUpAction,
		Usage:    "Migration action (up, down, redo, status, etc.)",
		Required: false,
	}
)

func init() {
	err := godotenv.Load(strings.Split(envPath, ",")...)
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctn, err := container.NewContainer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	app := cli.NewApp()
	app.Name = appName
	flags := []cli.Flag{}
	app.Metadata = map[string]any{
		config.FlagContainer: ctn,
	}
	app.Commands = []*cli.Command{
		{
			Name:    "api",
			Aliases: []string{},
			Usage:   "Run the api",
			Action:  startAPIServer,
			Flags:   append(flags, &configAddressFlag),
		},
		{
			Name:  "migrate",
			Usage: "Run database migrations (goose)",
			Flags: []cli.Flag{
				&configMigrateActionFlag,
			},
			Action: func(c *cli.Context) error {
				action := c.String(config.FlagMigrateAction)
				if action == "" {
					action = config.FlagUpAction
				}
				return migrate.Run(c.Context, cfg.Database, migrate.Command(action))
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
