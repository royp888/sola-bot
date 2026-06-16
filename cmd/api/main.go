package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/bootstrap"
	"github.com/dabowin/sola/internal/service"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	resources, err := bootstrap.New(ctx, "")
	if err != nil {
		log.Fatal(err)
	}
	defer resources.Close(context.Background())

	api.WarnIfPlaintextPassword(resources.Config.App.AdminPasswordHash, resources.Config.App.AdminPassword)
	deps := service.NewAPIDependencies(resources.Config, resources.Store)
	router := api.NewRouter(deps)
	server := &http.Server{
		Addr:              resources.Config.App.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("api listening on %s", resources.Config.App.HTTPAddr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatal(err)
		}
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}
}
