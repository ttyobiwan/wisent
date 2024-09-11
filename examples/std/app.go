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
		slog.Info("Shutting down")
		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
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

		// Allocate and do some work to test the benchmark
		size := 1024 * 1024 // 1MB
		largeSlice := make([]byte, size)
		for i := 0; i < size; i++ {
			largeSlice[i] = byte(i % 256)
		}
		sum := 0
		for _, b := range largeSlice {
			sum += int(b)
		}

		response := fmt.Sprintf("Hello, %s!", body.Name)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(response))
	})

	return &http.Server{
		Addr:         ":" + port,
		Handler:      a.loggingMiddleware(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}

func (a *app) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		slog.Info("Request started", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
		slog.Info("Request completed", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
