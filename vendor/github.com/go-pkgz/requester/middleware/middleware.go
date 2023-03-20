// Package middleware provides middlewares for htt.Client as RoundTripperHandler
package middleware

import (
	"net/http"
)

//go:generate moq -out mocks/repeater.go -pkg mocks -skip-ensure -fmt goimports . RepeaterSvc
//go:generate moq -out mocks/circuit_breaker.go -pkg mocks -skip-ensure -fmt goimports . CircuitBreakerSvc
//go:generate moq -out mocks/logger.go -pkg mocks -skip-ensure -fmt goimports logger Service:LoggerSvc
//go:generate moq -out mocks/cache.go -pkg mocks -skip-ensure -fmt goimports cache Service:CacheSvc

// RoundTripperHandler is a type for middleware handler
type RoundTripperHandler func(http.RoundTripper) http.RoundTripper

// RoundTripperFunc is a functional adapter for RoundTripperHandler
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip adopts function to the type
func (rt RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return rt(r) }
