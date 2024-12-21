package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"otp-core/internal/config"
	"time"

	"github.com/samber/do"
	"github.com/urfave/cli/v2"
)

func startAPIServer(c *cli.Context) error {
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	container, ok := c.App.Metadata[config.FlagContainer].(*do.Injector)
	if !ok {
		return errors.New("invalid service container")
	}

	router, err := do.Invoke[http.Handler](container)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    c.String(config.FlagAddress),
		Handler: router,
	}

	quit := make(chan os.Signal, 1)

	go func() {
		log.Printf("ListenAndServe: %s\n", c.String(config.FlagAddress))
		err := srv.ListenAndServe()
		if err != nil {
			log.Printf("ListenAndServe failed: %s\n", err)
			quit <- os.Kill
		}
	}()

	signal.Notify(quit, os.Interrupt)
	<-quit

	return srv.Shutdown(ctxShutdown)
}
