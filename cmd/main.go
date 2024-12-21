package main

import (
	"log"
	"os"
	"otp-core/internal/config"
	"otp-core/internal/content/container"
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
		Value:    "up",
		Usage:    "Configuration up or down in migration",
		Required: true,
	}
)

func init() {
	err := godotenv.Load(strings.Split(envPath, ",")...)
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	cfg, err := config.Load(os.Getenv("CONFIG_PATH"))
	if err != nil {
		log.Fatal(err)
	}
	ctn := container.NewContainer(cfg)

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
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
