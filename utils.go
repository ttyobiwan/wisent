package wisent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

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

func HealthCheckReadinessProbe(url string, client *http.Client) ReadinessProbe {
	return func(ctx context.Context) error {
		if client == nil {
			client = http.DefaultClient
		}

		startTime := time.Now()
		timeout := 5 * time.Second

		for {
			req, err := http.NewRequestWithContext(
				ctx,
				http.MethodGet,
				url,
				nil,
			)
			if err != nil {
				return fmt.Errorf("creating request: %w", err)
			}

			resp, err := client.Do(req)
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
