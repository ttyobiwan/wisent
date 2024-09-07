package main

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/ttyobiwan/wisent"
)

func TestHelloEndpoint(t *testing.T) {
	a := &app{os.Getenv}
	w := wisent.New("http://127.0.0.1:8080", a.start, "/health", nil)

	w.Test(t, []wisent.Test{
		{
			Name:    "POST hello 200",
			Request: w.NewRequest("POST", "/hello", strings.NewReader(`{"name": "World"}`)),
			AssertResponse: func(resp *http.Response, err error) {
				w.AssertResponseError(t, err)
				w.AssertResponseStatusCode(t, http.StatusOK, resp)
				w.AssertResponseBody(t, "Hello, World!", resp)
			},
			PreRequest:  func(req *http.Request) {},
			PostRequest: func(resp *http.Response) {},
		},
		{
			Name:    "POST hello 400",
			Request: w.NewRequest("POST", "/hello", nil),
			AssertResponse: func(resp *http.Response, err error) {
				w.AssertResponseError(t, err)
				w.AssertResponseStatusCode(t, http.StatusBadRequest, resp)
			},
			PreRequest:  func(req *http.Request) {},
			PostRequest: func(resp *http.Response) {},
		},
	})
}

func BenchmarkHelloEndpoint(b *testing.B) {
	a := &app{os.Getenv}
	w := wisent.New("http://127.0.0.1:8080", a.start, "/health", nil)

	w.Benchmark(b, wisent.Benchmark{
		Request: w.NewRequest("POST", "/hello", strings.NewReader(`{"name": "World"}`)),
		AssertResponse: func(resp *http.Response, err error) {
			w.AssertResponseError(b, err)
			w.AssertResponseStatusCode(b, http.StatusOK, resp)
			w.AssertResponseBody(b, "Hello, World!", resp)
		},
		PreRequest:  func(req *http.Request) {},
		PostRequest: func(resp *http.Response) {},
	})
}
