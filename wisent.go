package wisent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
)

type WisentOpt func(w *Wisent)

func WithStartFunc(start StartFunc) WisentOpt { return func(w *Wisent) { w.Start = start } }

func WithReadinessProbe(rp ReadinessProbe) WisentOpt {
	return func(w *Wisent) { w.ReadinessProbe = rp }
}

func WithHttpClient(client *http.Client) WisentOpt {
	return func(w *Wisent) { w.HttpClient = client }
}

func WithRequestWrapper(rw RequestWrapper) WisentOpt {
	return func(w *Wisent) { w.RequestWrapper = rw }
}

func WithLogger(logger *slog.Logger) WisentOpt {
	return func(w *Wisent) { w.Logger = logger }
}

// Wisent represents a configuration for running API tests and benchmarks.
// It provides a flexible way to set up and execute HTTP requests against a target API.
type Wisent struct {
	// BaseURL specifies base url for all requests.
	BaseURL string
	// Start is a function that starts the application under test.
	// It returns a shutdown function that will be called when the test is complete.
	// If Start is not provided, the test will run against an already running application.
	Start StartFunc
	// ReadinessProbe is a function that checks if the application is ready to receive requests.
	// It should block until the application is ready.
	// If empty, no readiness probe is done.
	ReadinessProbe ReadinessProbe
	// HttpClient is the HTTP client used to make requests.
	// If not provided, a default client will be used.
	HttpClient *http.Client
	// RequestWrapper is a function that wraps the HTTP request.
	// It can be used to modify the logic around making request.
	// An example could be applying retry policy.
	// If empty, only HttpClient.Do will be called.
	RequestWrapper RequestWrapper
	// Logger is used for logging test progress and information.
	// If not provided, a default logger writing to io.Discard will be used.
	Logger *slog.Logger
}

// New creates and returns a new Wisent instance with the specified base URL and options.
// It applies the provided options to customize the Wisent instance.
func New(baseUrl string, options ...WisentOpt) *Wisent {
	w := &Wisent{BaseURL: baseUrl}
	for _, opt := range options {
		opt(w)
	}
	if w.HttpClient == nil {
		w.HttpClient = DefaultHttpClient()
	}
	if w.Logger == nil {
		w.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return w
}

// NewRequest is a helper method that allows building requests without checking for errors.
// This is handy in tests, where we (usually) know what we are doing.
func (w *Wisent) NewRequest(method string, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, w.BaseURL+url, body)
	if err != nil {
		panic(fmt.Errorf("creating request: %v", err))
	}
	return req
}

// Test runs a series of tests against the configured API.
// It takes a testing.T instance and a slice of Test structs.
// For each Test, it executes the HTTP request and runs the associated assertions.
func (w *Wisent) Test(t *testing.T, tests []Test) error {
	w.Logger.Info("Starting tests")
	ctx, cancel := context.WithCancel(context.Background())

	if w.Start != nil {
		w.Logger.Info("Starting the app")
		shutdown := w.Start(ctx)
		defer func() {
			w.Logger.Info("Shutting down")
			cancel()
			shutdown(context.Background())
		}()
	} else {
		defer cancel()
	}

	if w.ReadinessProbe != nil {
		w.Logger.Info("Starting the readiness probe")
		w.ReadinessProbe(ctx, w)
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			w.Logger.Info("Running the test", "name", tt.Name)

			if tt.PreRequest != nil {
				tt.PreRequest(tt.Request)
			}

			var resp *http.Response
			var err error
			if w.RequestWrapper != nil {
				resp, err = w.RequestWrapper(w, tt.Request)
			} else {
				w.Logger.Info("Performing the request")
				resp, err = w.HttpClient.Do(tt.Request)
			}

			if tt.PostRequest != nil {
				tt.PostRequest(resp)
			}

			tt.AssertResponse(resp, err)

			resp.Body.Close()
			w.Logger.Info("Finished test", "name", tt.Name)
		})
	}

	w.Logger.Info("Testing done")
	return nil
}

// Benchmark runs a benchmark test against the configured API.
// It takes a testing.B instance and a Benchmark struct.
// For each iteration, it executes the HTTP request and runs the associated assertions.
// The benchmark measures the performance of the API under test.
func (w *Wisent) Benchmark(b *testing.B, bm Benchmark) error {
	w.Logger.Info("Starting the benchmark")
	ctx, cancel := context.WithCancel(context.Background())

	if w.Start != nil {
		w.Logger.Info("Starting the app")

		shutdown := w.Start(ctx)
		defer func() {
			w.Logger.Info("Shutting down")
			cancel()
			shutdown(ctx)
		}()

	} else {
		defer cancel()
	}

	if w.ReadinessProbe != nil {
		w.Logger.Info("Starting the readiness probe")
		w.ReadinessProbe(ctx, w)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Logger.Info("Running the benchmark")

		req := bm.RequestF()

		if bm.PreRequest != nil {
			bm.PreRequest(req)
		}

		var resp *http.Response
		var err error
		if w.RequestWrapper != nil {
			resp, err = w.RequestWrapper(w, req)
		} else {
			w.Logger.Info("Performing the request")
			resp, err = w.HttpClient.Do(req)
		}

		if bm.PostRequest != nil {
			bm.PostRequest(resp)
		}

		bm.AssertResponse(resp, err)

		resp.Body.Close()
		w.Logger.Info("Finished benchmark")
	}

	w.Logger.Info("Benchmarking done")
	return nil
}

// BenchmarkParallel runs a parallel benchmark test against the configured API.
// It takes a testing.B instance and a Benchmark struct.
// For each goroutine, it repeatedly executes the HTTP request and runs the associated assertions.
// The benchmark measures the performance of the API under test in a concurrent scenario.
// This method is suitable for simulating high concurrency and measuring how the API performs under parallel load.
func (w *Wisent) BenchmarkParallel(b *testing.B, bm Benchmark) error {
	w.Logger.Info("Starting the parallel benchmark")
	ctx, cancel := context.WithCancel(context.Background())

	if w.Start != nil {
		w.Logger.Info("Starting the app")

		shutdown := w.Start(ctx)
		defer func() {
			w.Logger.Info("Shutting down")
			cancel()
			shutdown(context.Background())
		}()
	} else {
		defer cancel()
	}

	if w.ReadinessProbe != nil {
		w.Logger.Info("Starting the readiness probe")
		w.ReadinessProbe(ctx, w)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w.Logger.Info("Running the benchmark")

			req := bm.RequestF()

			if bm.PreRequest != nil {
				bm.PreRequest(req)
			}

			var resp *http.Response
			var err error
			if w.RequestWrapper != nil {
				resp, err = w.RequestWrapper(w, req)
			} else {
				w.Logger.Info("Performing the request")
				resp, err = w.HttpClient.Do(req)
			}

			if bm.PostRequest != nil {
				bm.PostRequest(resp)
			}

			bm.AssertResponse(resp, err)

			resp.Body.Close()
			w.Logger.Info("Finished benchmark")
		}
	})

	w.Logger.Info("Benchmarking done")
	return nil
}

// AssertResponseError is a testing helper method that checks if response error is empty.
func (w *Wisent) AssertResponseError(tb testing.TB, err error) {
	if err != nil {
		tb.Fatalf("Error performing the request: %v", err)
	}
}

// AssertResponseStatusCode is a testing helper method that compares response status code.
func (w *Wisent) AssertResponseStatusCode(tb testing.TB, expected int, resp *http.Response) {
	if resp.StatusCode != expected {
		tb.Fatalf("Incorrect status code, got: %v, want: %v", resp.StatusCode, expected)
	}
}

// AssertResponseBody is a testing helper method that compares response body.
func (w *Wisent) AssertResponseBody(tb testing.TB, expected string, resp *http.Response) {
	actualBody, err := io.ReadAll(resp.Body)
	if err != nil {
		tb.Fatalf("Error reading response body: %v", err)
	}

	if string(actualBody) != expected {
		tb.Fatalf("Body mismatch\nExpected: %s\nActual: %s", expected, actualBody)
	}
}
