// Package requester wraps http.Client with a chain of middleware.RoundTripperHandler.
// Each RoundTripperHandler implements a part of functionality expanding http.Request oar altering
// the flow in some way. Some middlewares set headers, some add logging and caching, some limit concurrency.
// User can provide custom middlewares.
package requester

import (
	"net/http"

	"github.com/go-pkgz/requester/middleware"
)

// Requester provides a wrapper for the standard http.Do request.
type Requester struct {
	client      http.Client
	middlewares []middleware.RoundTripperHandler
}

// New creates requester with defaults
func New(client http.Client, middlewares ...middleware.RoundTripperHandler) *Requester {
	return &Requester{
		client:      client,
		middlewares: middlewares,
	}
}

// Use adds middleware(s) to the requester chain
func (r *Requester) Use(middlewares ...middleware.RoundTripperHandler) {
	r.middlewares = append(r.middlewares, middlewares...)
}

// With makes a new Requested with inherited middlewares and add passed middleware(s) to the chain
func (r *Requester) With(middlewares ...middleware.RoundTripperHandler) *Requester {
	res := &Requester{
		client:      r.client,
		middlewares: append(r.middlewares, middlewares...),
	}
	return res
}

// Client returns http.Client with all middlewares injected
func (r *Requester) Client() *http.Client {
	cl := r.client
	if cl.Transport == nil {
		cl.Transport = http.DefaultTransport
	}
	for _, handler := range r.middlewares {
		cl.Transport = handler(cl.Transport)
	}
	return &cl
}

// Do runs http request with optional middleware handlers wrapping the request
func (r *Requester) Do(req *http.Request) (*http.Response, error) {
	return r.Client().Do(req)
}
