package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type app struct {
	getenv func(string) string
}

func (a *app) start(ctx context.Context) (shutdown func(ctx context.Context)) {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	// Start server
	server := a.getServer()
	go func() {
		slog.Info("Starting server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Error listening", "error", err)
		}
	}()

	// Return shutdown
	return func(ctx context.Context) {
		<-ctx.Done()

		slog.Info("Shutting down")
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error shutting down", "error", err)
		}

		slog.Info("Server shut down")
	}
}

func (a *app) getServer() *http.Server {
	port := a.getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("POST /hello", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("/hello")

		var body struct {
			Name string `json:"name"`
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		response := fmt.Sprintf("Hello, %s!", body.Name)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(response))
	})

	return &http.Server{Addr: ":" + port, Handler: mux}
}
