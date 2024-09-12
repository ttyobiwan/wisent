package wisent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ErrHealthCheckTimeout is returned when the health check fails to complete within the specified timeout period.
var ErrHealthCheckTimeout = errors.New("health check timeout reached")

// DefaultHttpClient returns a pre-configured http.Client with specific timeout and connection settings.
// This client is suitable for making HTTP requests with consistent timeout behavior and connection reuse.
func DefaultHttpClient() *http.Client {
	return &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   3 * time.Second,
				KeepAlive: 10 * time.Second,
			}).Dial,
			ResponseHeaderTimeout: 3 * time.Second,
			ExpectContinueTimeout: 3 * time.Second,
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       3 * time.Second,
		},
	}
}

// HealthCheckReadinessProbe creates a ReadinessProbe function that performs HTTP health checks.
//
// It sends GET requests to the specified URL at regular intervals until either:
// - A successful response (HTTP 200 OK) is received
// - The context is cancelled
// - The specified timeout duration is reached
//
// The returned function will return nil if the health check succeeds, or an error
// if the health check fails due to timeout or context cancellation.
func HealthCheckReadinessProbe(url string, timeout time.Duration, sleep time.Duration) ReadinessProbe {
	return func(ctx context.Context, w *Wisent) error {
		startTime := time.Now()
		for {
			w.Logger.Info("Checking readiness")
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
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					if time.Since(startTime) >= timeout {
						return ErrHealthCheckTimeout
					}
					time.Sleep(sleep)
					continue
				}
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
					return ErrHealthCheckTimeout
				}
				time.Sleep(sleep)
			}
		}
	}
}

// SimpleRetry creates a RequestWrapper that implements a simple retry mechanism for HTTP requests.
//
// It attempts to perform the request up to 'maxAttempts' times, with an increasing delay between each attempt.
// The delay starts at 'baseSleep' and increases linearly with each retry.
//
// The wrapper logs each attempt and any errors encountered. If all attempts fail, it returns the last error encountered.
func SimpleRetry(maxAttempts int, baseSleep time.Duration) RequestWrapper {
	return func(w *Wisent, req *http.Request) (resp *http.Response, err error) {
		for i := range 5 {
			w.Logger.Info("Performing the request")
			resp, err = w.HttpClient.Do(req)
			if err != nil {
				w.Logger.Warn("Error performing request, sleeping", "err", err, "sleep", time.Duration(i*int(baseSleep)))
				time.Sleep(time.Duration(i * int(baseSleep)))
				continue
			}
			return resp, err
		}
		return nil, err
	}
}
