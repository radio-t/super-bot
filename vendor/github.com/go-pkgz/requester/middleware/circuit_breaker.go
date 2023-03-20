package middleware

import (
	"fmt"
	"net/http"
)

// CircuitBreaker middleware injects external CircuitBreakerSvc into the call chain
func CircuitBreaker(svc CircuitBreakerSvc) RoundTripperHandler {

	return func(next http.RoundTripper) http.RoundTripper {
		fn := func(req *http.Request) (*http.Response, error) {

			if svc == nil {
				return next.RoundTrip(req)
			}

			resp, e := svc.Execute(func() (interface{}, error) {
				return next.RoundTrip(req)
			})
			if e != nil {
				return nil, fmt.Errorf("circuit breaker: %w", e)
			}
			return resp.(*http.Response), nil
		}
		return RoundTripperFunc(fn)
	}
}

// CircuitBreakerSvc is an interface wrapping any function to send a request with circuit breaker.
// can be used with github.com/sony/gobreaker or any similar implementations
type CircuitBreakerSvc interface {
	Execute(req func() (interface{}, error)) (interface{}, error)
}

// CircuitBreakerFunc is an adapter to allow the use of ordinary functions as CircuitBreakerSvc.
type CircuitBreakerFunc func(req func() (interface{}, error)) (interface{}, error)

// Execute CircuitBreakerFunc
func (c CircuitBreakerFunc) Execute(req func() (interface{}, error)) (interface{}, error) {
	return c(req)
}
