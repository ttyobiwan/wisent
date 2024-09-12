# 🦬 wisent

Wisent is a Go library designed for testing and benchmarking APIs. It provides a flexible and easy-to-use framework for setting up, running, and asserting HTTP-based API tests and benchmarks.

Wisent has no external dependecies. It uses Go standard library and native benchmarking.

## Features

- Simple API for defining and running tests and benchmarks
- Support for starting and stopping the application under test
- Customizable HTTP client and request wrapper
- Built-in assertions for common HTTP response checks
- Parallel benchmarking capabilities
- Configurable logging
- Readiness probe functionality

## Installation

To install Wisent, use `go get`:

```
go get github.com/ttyobiwan/wisent
```

## Quick Start

To run tests and benchmarks, use Go native commands, like `go test ./...` or `go test ./... -bench=.`.

### Testing

Here's a simple example of how to use Wisent for testing an API endpoint:

```go
func TestHelloEndpoint(t *testing.T) {
    w := wisent.New("http://127.0.0.1:8080")

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
    })
}
```

### Sequential Benchmarking

Use the `Benchmark` method to run sequential benchmarks:

```go
func BenchmarkHelloEndpoint(b *testing.B) {
    w := wisent.New("http://127.0.0.1:8080")

    w.Benchmark(b, wisent.Benchmark{
        RequestF: func() *http.Request {
            return w.NewRequest("POST", "/hello", strings.NewReader(`{"name": "World"}`))
        },
        AssertResponse: func(resp *http.Response, err error) {
            w.AssertResponseError(b, err)
            w.AssertResponseStatusCode(b, http.StatusOK, resp)
            w.AssertResponseBody(b, "Hello, World!", resp)
        },
    })
}
```

### Parallel Benchmarking

For testing how your API performs under concurrent load, use the `BenchmarkParallel` method:

```go
func BenchmarkParallelHelloEndpoint(b *testing.B) {
    w := wisent.New("http://127.0.0.1:8080")

    w.BenchmarkParallel(b, wisent.Benchmark{
        RequestF: func() *http.Request {
            return w.NewRequest("POST", "/hello", strings.NewReader(`{"name": "World"}`))
        },
        AssertResponse: func(resp *http.Response, err error) {
            w.AssertResponseError(b, err)
            w.AssertResponseStatusCode(b, http.StatusOK, resp)
            w.AssertResponseBody(b, "Hello, World!", resp)
        },
    })
}
```

### Customization

Wisent allows you to customize various aspects of your benchmarks:

- **Pre-request hooks**: Execute actions before each request
- **Post-request hooks**: Perform operations after each request
- **Custom HTTP clients**: Use your own HTTP client for specific needs
- **Request wrappers**: Modify requests or add retry logic

Example with customizations:

```go
w := wisent.New(
    "http://127.0.0.1:8080",
    wisent.WithStartFunc(a.start),
    wisent.WithReadinessProbe(wisent.HealthCheckReadinessProbe("/health", 5*time.Second, 100*time.Millisecond)),
    wisent.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, nil))),
    wisent.WithRequestWrapper(wisent.SimpleRetry(3, 100*time.Millisecond)),
)

w.BenchmarkParallel(b, wisent.Benchmark{
    RequestF: func() *http.Request {
        return w.NewRequest("POST", "/hello", strings.NewReader(`{"name": "World"}`))
    },
    AssertResponse: func(resp *http.Response, err error) {
        w.AssertResponseError(b, err)
        w.AssertResponseStatusCode(b, http.StatusOK, resp)
        w.AssertResponseBody(b, "Hello, World!", resp)
    },
    PreRequest:  func(req *http.Request) { slog.Info("Making request") },
    PostRequest: func(resp *http.Response) { slog.Info("Done making request") },
})
```

See the examples in the `examples` directory for more advanced usage patterns.
