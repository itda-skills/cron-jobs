package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/itda-skills/cron-jobs/internal/app"
	"github.com/itda-skills/cron-jobs/internal/httpapi"
	"github.com/itda-skills/cron-jobs/internal/webui"
)

func main() {
	settings := app.LoadSettingsFromEnv()

	if err := settings.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid settings: %v\n", err)
		os.Exit(1)
	}

	service := app.NewService(settings)
	if err := service.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "load service: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go service.Start(ctx)

	server := &http.Server{
		Addr:    settings.Addr,
		Handler: webui.Server{Service: service}.Routes(httpapi.Server{Service: service}.Routes()),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), app.ShutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("cron-jobs listening on %s with config %s\n", settings.Addr, settings.ConfigPath)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "http server: %v\n", err)
		os.Exit(1)
	}
}
