package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dabowin/sola/internal/bootstrap"
	"github.com/dabowin/sola/internal/worker"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	resources, err := bootstrap.New(ctx, "")
	if err != nil {
		log.Fatal(err)
	}
	defer resources.Close(context.Background())

	runner := worker.New(resources.Config, resources.Store, resources.Logger)
	if err := runner.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
