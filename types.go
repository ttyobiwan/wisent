package wisent

import (
	"context"
	"net/http"
)

type (
	StartFunc      func(context.Context) func(context.Context)
	ReadinessProbe func(context.Context) error
)

type Test struct {
	Name           string
	Request        *http.Request
	PreRequest     func(req *http.Request)
	AssertResponse func(resp *http.Response, err error)
	PostRequest    func(resp *http.Response)
}

type Benchmark struct {
	Request        *http.Request
	PreRequest     func(req *http.Request)
	AssertResponse func(resp *http.Response, err error)
	PostRequest    func(resp *http.Response)
}
