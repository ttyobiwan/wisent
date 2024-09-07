package main

import (
	"context"
	"log/slog"
	"os"
)

func run(ctx context.Context, getenv func(string) string) error {
	if getenv("DEBUG") != "true" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}

	a := app{getenv}
	shutdown := a.start(ctx)
	defer shutdown(ctx)

	return nil
}

func main() {
	if err := run(context.Background(), os.Getenv); err != nil {
		slog.Error("Error starting the server", "error", err)
	}
}
