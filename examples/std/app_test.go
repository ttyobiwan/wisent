package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ttyobiwan/wisent"
)

func TestHelloEndpoint(t *testing.T) {
	a := &app{os.Getenv}
	w := wisent.New(
		"http://127.0.0.1:8080",
		wisent.WithStartFunc(a.start),
		wisent.WithReadinessProbe(wisent.HealthCheckReadinessProbe("/health", 5*time.Second, 100*time.Millisecond)),
	)

	w.Test(t, []wisent.Test{
		{
			Name:    "POST hello 200",
			Request: w.NewRequest("POST", "/hello", strings.NewReader(`{"name": "World"}`)),
			AssertResponse: func(resp *http.Response, err error) {
				w.AssertResponseError(t, err)
				w.AssertResponseStatusCode(t, http.StatusOK, resp)
				w.AssertResponseBody(t, "Hello, World!", resp)
			},
		},
		{
			Name:    "POST hello 400",
			Request: w.NewRequest("POST", "/hello", nil),
			AssertResponse: func(resp *http.Response, err error) {
				w.AssertResponseError(t, err)
				w.AssertResponseStatusCode(t, http.StatusBadRequest, resp)
			},
		},
	})
}

func BenchmarkHelloEndpoint(b *testing.B) {
	a := &app{os.Getenv}
	w := wisent.New(
		"http://127.0.0.1:8080",
		wisent.WithStartFunc(a.start),
		wisent.WithReadinessProbe(wisent.HealthCheckReadinessProbe("/health", 5*time.Second, 100*time.Millisecond)),
	)

	w.Benchmark(b, wisent.Benchmark{
		RequestF: func() *http.Request { return w.NewRequest("POST", "/hello", strings.NewReader(`{"name": "World"}`)) },
		AssertResponse: func(resp *http.Response, err error) {
			w.AssertResponseError(b, err)
			w.AssertResponseStatusCode(b, http.StatusOK, resp)
			w.AssertResponseBody(b, "Hello, World!", resp)
		},
	})
}

func BenchmarkParallelHelloEndpoint(b *testing.B) {
	a := &app{os.Getenv}
	slog.SetLogLoggerLevel(slog.LevelError)

	w := wisent.New(
		"http://127.0.0.1:8080",
		wisent.WithStartFunc(a.start),
		wisent.WithReadinessProbe(wisent.HealthCheckReadinessProbe("/health", 5*time.Second, 100*time.Millisecond)),
		wisent.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, nil))),
		wisent.WithRequestWrapper(wisent.SimpleRetry(3, 100*time.Millisecond)),
	)

	w.BenchmarkParallel(b, wisent.Benchmark{
		RequestF: func() *http.Request { return w.NewRequest("POST", "/hello", strings.NewReader(`{"name": "World"}`)) },
		AssertResponse: func(resp *http.Response, err error) {
			w.AssertResponseError(b, err)
			w.AssertResponseStatusCode(b, http.StatusOK, resp)
			w.AssertResponseBody(b, "Hello, World!", resp)
		},
		PreRequest:  func(req *http.Request) { slog.Info("Making request") },
		PostRequest: func(resp *http.Response) { slog.Info("Done making request") },
	})
}
