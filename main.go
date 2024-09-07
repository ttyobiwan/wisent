package wisent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

type (
	StartFunc       func(context.Context) func(context.Context)
	HealthCheckFunc func(context.Context) error
)

type Wisent struct {
	BaseURL     string
	Start       StartFunc
	HealthCheck HealthCheckFunc
	HttpClient  *http.Client
}

func New(baseUrl string, start StartFunc, healthcheck string, httpClient *http.Client) *Wisent {
	w := &Wisent{BaseURL: baseUrl, Start: start, HealthCheck: nil, HttpClient: nil}
	w.HealthCheck = w.DefaultHealthCheck(healthcheck)
	if httpClient == nil {
		w.HttpClient = w.DefaultHttpClient()
	}
	return w
}

func (w *Wisent) DefaultHealthCheck(url string) HealthCheckFunc {
	return func(ctx context.Context) error {
		startTime := time.Now()
		timeout := 5 * time.Second

		for {
			req, err := http.NewRequestWithContext(
				ctx,
				http.MethodGet,
				w.BaseURL+url,
				nil,
			)
			if err != nil {
				return fmt.Errorf("creating request: %w", err)
			}

			resp, err := w.HttpClient.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return nil
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if time.Since(startTime) >= timeout {
					return errors.New("timeout reached when waiting for readiness")
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

func (w *Wisent) DefaultHttpClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       10 * time.Second,
		},
	}
}

func (w *Wisent) NewRequest(method string, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, w.BaseURL+url, body)
	if err != nil {
		panic(fmt.Errorf("creating request: %v", err))
	}
	return req
}

func (w *Wisent) Test(t *testing.T, tests []Test) error {
	ctx, cancel := context.WithCancel(context.Background())
	shutdown := w.Start(ctx)
	defer func() {
		cancel()
		shutdown(ctx)
	}()

	w.HealthCheck(ctx)

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			tt.PreRequest(tt.Request)
			resp, err := w.HttpClient.Do(tt.Request)
			tt.PostRequest(resp)
			tt.AssertResponse(resp, err)
		})
	}

	return nil
}

func (w *Wisent) Benchmark(b *testing.B, bm Benchmark) error {
	ctx, cancel := context.WithCancel(context.Background())
	shutdown := w.Start(ctx)
	defer func() {
		cancel()
		shutdown(ctx)
	}()

	w.HealthCheck(ctx)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bm.PreRequest(bm.Request)
		resp, err := w.HttpClient.Do(bm.Request)
		bm.PostRequest(resp)
		bm.AssertResponse(resp, err)
	}

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
		tb.Errorf("Error reading response body: %v", err)
	}

	if string(actualBody) != expected {
		tb.Fatalf("Body mismatch\nExpected: %s\nActual: %s", expected, actualBody)
	}
}
