package wisent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

type Wisent struct {
	BaseURL        string
	Start          StartFunc
	ReadinessProbe ReadinessProbe
	HttpClient     *http.Client
	Logger         *slog.Logger
}

type WisentOpt func(w *Wisent)

func WithStartFunc(start StartFunc) WisentOpt { return func(w *Wisent) { w.Start = start } }

func WithReadinessProbe(rp ReadinessProbe) WisentOpt {
	return func(w *Wisent) { w.ReadinessProbe = rp }
}

func WithHttpClient(client *http.Client) WisentOpt {
	return func(w *Wisent) { w.HttpClient = client }
}

func WithLogger(logger *slog.Logger) WisentOpt {
	return func(w *Wisent) { w.Logger = logger }
}

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

func (w *Wisent) NewRequest(method string, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, w.BaseURL+url, body)
	if err != nil {
		panic(fmt.Errorf("creating request: %v", err))
	}
	return req
}

func (w *Wisent) Test(t *testing.T, tests []Test) error {
	w.Logger.Info("Starting tests")
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
		w.ReadinessProbe(ctx)
	}

	for _, tt := range tests {
		w.Logger.Info("Running the test", "name", tt.Name)
		t.Run(tt.Name, func(t *testing.T) {
			if tt.PreRequest != nil {
				tt.PreRequest(tt.Request)
			}
			w.Logger.Info("Performing the request")
			resp, err := w.HttpClient.Do(tt.Request)
			if tt.PostRequest != nil {
				tt.PostRequest(resp)
			}
			tt.AssertResponse(resp, err)
		})
		w.Logger.Info("Finished test", "name", tt.Name)
	}

	w.Logger.Info("Testing done")
	return nil
}

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
		w.ReadinessProbe(ctx)
	}

	b.ResetTimer()

	maxIter := bm.MaxIter
	if maxIter == 0 {
		maxIter = b.N
	}
	for i := 0; i < b.N && i < maxIter; i++ { // TODO: Test parallel
		w.Logger.Info("Running the benchmark")
		req := bm.RequestF()
		if bm.PreRequest != nil {
			bm.PreRequest(req)
		}

		var resp *http.Response
		var err error
		for j := range 5 { // TODO: Custom strat
			w.Logger.Info("Performing the request")
			resp, err = w.HttpClient.Do(req)
			if err == nil {
				break
			}
			time.Sleep(time.Duration(j * j * 100 * int(time.Millisecond)))
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

func (w *Wisent) AssertResponseError(tb testing.TB, err error) {
	if err != nil {
		tb.Fatalf("Error performing the request: %v", err)
	}
}

func (w *Wisent) AssertResponseStatusCode(tb testing.TB, expected int, resp *http.Response) {
	if resp.StatusCode != expected {
		tb.Fatalf("Incorrect status code, got: %v, want: %v", resp.StatusCode, expected)
	}
}

func (w *Wisent) AssertResponseBody(tb testing.TB, expected string, resp *http.Response) {
	actualBody, err := io.ReadAll(resp.Body)
	if err != nil {
		tb.Fatalf("Error reading response body: %v", err)
	}

	if string(actualBody) != expected {
		tb.Fatalf("Body mismatch\nExpected: %s\nActual: %s", expected, actualBody)
	}
}
