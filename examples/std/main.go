package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
)

func run(ctx context.Context, getenv func(string) string) error {
	if getenv("DEBUG") != "true" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}

	a := app{getenv}
	shutdown := a.start(ctx)
	defer shutdown(ctx)
	<-ctx.Done()

	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx, os.Getenv); err != nil {
		slog.Error("Error starting the server", "error", err)
	}
}
