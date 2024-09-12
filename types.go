package wisent

import (
	"context"
	"net/http"
)

type (
	// StartFunc is a function that starts a process and returns a shutdown function.
	// It takes a context.Context as input and returns a function that can be called to shut down the process.
	StartFunc func(context.Context) func(context.Context)
	// ReadinessProbe is a function that checks if a Wisent instance is ready.
	// It takes a context.Context and a pointer to a Wisent instance as input and returns an error if the instance is not ready.
	ReadinessProbe func(context.Context, *Wisent) error
	// RequestWrapper is a function that wraps an HTTP request.
	// It takes a pointer to a Wisent instance and an *http.Request as input and returns an *http.Response and an error.
	RequestWrapper func(w *Wisent, req *http.Request) (*http.Response, error)
)

// Test represents a test case for a Wisent instance.
// It includes a name, an HTTP request, optional pre and post request functions, and a function to assert the response.
type Test struct {
	Name           string
	Request        *http.Request
	PreRequest     func(req *http.Request)
	AssertResponse func(resp *http.Response, err error)
	PostRequest    func(resp *http.Response)
}

// Benchmark represents a benchmark test for a Wisent instance.
// It includes functions to generate requests, optionally modify them before sending,
// assert responses, and perform post-request actions.
type Benchmark struct {
	RequestF       func() *http.Request
	PreRequest     func(req *http.Request)
	AssertResponse func(resp *http.Response, err error)
	PostRequest    func(resp *http.Response)
}
