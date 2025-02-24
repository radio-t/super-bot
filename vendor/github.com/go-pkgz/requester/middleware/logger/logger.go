// Package logger implements middleware for request logging.
package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-pkgz/requester/middleware"
)

// Middleware for logging requests
type Middleware struct {
	Service
	prefix  string
	body    bool
	headers bool
}

// New creates logging middleware with optional parameters turning on logging elements
func New(svc Service, opts ...func(m *Middleware)) *Middleware {
	res := Middleware{Service: svc}
	for _, opt := range opts {
		opt(&res)
	}
	return &res
}

// Middleware request logging
func (m Middleware) Middleware(next http.RoundTripper) http.RoundTripper {
	fn := func(req *http.Request) (resp *http.Response, err error) {
		if m.Service == nil {
			return next.RoundTrip(req)
		}

		st := time.Now()
		logParts := []string{}
		if m.prefix != "" {
			logParts = append(logParts, m.prefix)
		}
		logParts = append(logParts, req.Method, req.URL.String()+",")

		headerLog := []byte{} // nolint
		if m.headers {
			if headerLog, err = json.Marshal(req.Header); err != nil {
				headerLog = []byte(fmt.Sprintf("headers: %v", req.Header))
			}
			logParts = append(logParts, string(headerLog)+",")
		}

		bodyLog := ""
		if m.body && req.Body != nil {
			body, e := io.ReadAll(req.Body)
			if e == nil {
				_ = req.Body.Close()
				req.Body = io.NopCloser(bytes.NewReader(body))
				bodyLog = " body: " + string(body)
				if len(bodyLog) > 1024 {
					bodyLog = bodyLog[:1024] + "..."
				}
				bodyLog = strings.Replace(bodyLog, "\n", " ", -1)
			}
		}
		if bodyLog != "" {
			logParts = append(logParts, bodyLog+",")
		}
		resp, err = next.RoundTrip(req)
		logParts = append(logParts, fmt.Sprintf("time: %v", time.Since(st)))
		m.Logf(strings.Join(logParts, " "))
		return resp, err
	}
	return middleware.RoundTripperFunc(fn)
}

// Prefix sets logging prefix for each line
func Prefix(prefix string) func(m *Middleware) {
	return func(m *Middleware) {
		m.prefix = prefix
	}
}

// WithBody enables body logging
func WithBody(m *Middleware) {
	m.body = true
}

// WithHeaders enables headers logging
func WithHeaders(m *Middleware) {
	m.headers = true
}

// Service defined logger interface used everywhere in the package
type Service interface {
	Logf(format string, args ...interface{})
}

// Func type is an adapter to allow the use of ordinary functions as Service.
type Func func(format string, args ...interface{})

// Logf calls f(id)
func (f Func) Logf(format string, args ...interface{}) { f(format, args...) }

// Std logger sends to std default logger directly
var Std = Func(func(format string, args ...interface{}) { log.Printf(format, args...) })
