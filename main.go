package wisent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type Wisent struct {
	BaseURL        string
	Start          StartFunc
	ReadinessProbe ReadinessProbe
	HttpClient     *http.Client
}

type WisentOpt func(w *Wisent)

func WithStartFunc(start StartFunc) WisentOpt { return func(w *Wisent) { w.Start = start } }

func WithReadinessProbe(rp ReadinessProbe) WisentOpt {
	return func(w *Wisent) { w.ReadinessProbe = rp }
}

func WithHttpClient(client *http.Client) WisentOpt {
	return func(w *Wisent) { w.HttpClient = client }
}

func New(baseUrl string, options ...WisentOpt) *Wisent {
	w := &Wisent{BaseURL: baseUrl}
	for _, opt := range options {
		opt(w)
	}
	if w.HttpClient == nil {
		w.HttpClient = DefaultHttpClient()
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
	ctx, cancel := context.WithCancel(context.Background())

	if w.Start != nil {
		shutdown := w.Start(ctx)
		defer func() {
			cancel()
			shutdown(ctx)
		}()
	} else {
		defer cancel()
	}

	if w.ReadinessProbe != nil {
		w.ReadinessProbe(ctx)
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if tt.PreRequest != nil {
				tt.PreRequest(tt.Request)
			}
			resp, err := w.HttpClient.Do(tt.Request)
			if tt.PostRequest != nil {
				tt.PostRequest(resp)
			}
			tt.AssertResponse(resp, err)
		})
	}

	return nil
}

func (w *Wisent) Benchmark(b *testing.B, bm Benchmark) error {
	ctx, cancel := context.WithCancel(context.Background())

	if w.Start != nil {
		shutdown := w.Start(ctx)
		defer func() {
			cancel()
			shutdown(ctx)
		}()
	} else {
		defer cancel()
	}

	if w.ReadinessProbe != nil {
		w.ReadinessProbe(ctx)
	}

	var bodyBuffer bytes.Buffer
	if bm.Request.Body != nil {
		defer bm.Request.Body.Close()
		_, err := io.Copy(&bodyBuffer, bm.Request.Body)
		if err != nil {
			return fmt.Errorf("reading request body: %w", err)
		}
	}

	b.ResetTimer()

	maxIterations := 10000 // TODO: Add that to bench
	for i := 0; i < b.N && i < maxIterations; i++ {
		req := copyRequest(ctx, bm.Request, bodyBuffer)

		if bm.PreRequest != nil {
			bm.PreRequest(req)
		}

		var resp *http.Response
		var err error
		for j := range 5 { // TODO: Custom strat
			resp, err = w.HttpClient.Do(req)
			if err == nil {
				break
			}
			time.Sleep(time.Duration(j * j * 100 * int(time.Millisecond))) // TODO: retryFor
		}

		if bm.PostRequest != nil {
			bm.PostRequest(resp)
		}

		bm.AssertResponse(resp, err)

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
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
		tb.Fatalf("Error reading response body: %v", err)
	}

	if string(actualBody) != expected {
		tb.Fatalf("Body mismatch\nExpected: %s\nActual: %s", expected, actualBody)
	}
}

func copyRequest(ctx context.Context, req *http.Request, body bytes.Buffer) *http.Request {
	req, err := http.NewRequestWithContext(ctx, req.Method, req.URL.String(), strings.NewReader(body.String()))
	if err != nil {
		panic(fmt.Errorf("copying request: %v", err))
	}
	for k, v := range req.Header {
		req.Header[k] = v
	}
	return req
}
