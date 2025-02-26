package middleware

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// BackoffType represents backoff strategy
type BackoffType int

const (
	// BackoffConstant is a backoff strategy with constant delay
	BackoffConstant BackoffType = iota
	// BackoffLinear is a backoff strategy with linear delay
	BackoffLinear
	// BackoffExponential is a backoff strategy with exponential delay
	BackoffExponential
)

// RetryMiddleware implements a retry mechanism for http requests with configurable backoff strategies.
// It supports constant, linear and exponential backoff with optional jitter for better load distribution.
// By default retries on network errors and 5xx responses. Can be configured to retry on specific status codes
// or to exclude specific codes from retry.
//
// Default configuration:
//   - 3 attempts
//   - Initial delay: 100ms
//   - Max delay: 30s
//   - Exponential backoff
//   - 10% jitter
//   - Retries on 5xx status codes
type RetryMiddleware struct {
	next         http.RoundTripper
	attempts     int
	initialDelay time.Duration
	maxDelay     time.Duration
	backoff      BackoffType
	jitterFactor float64
	retryCodes   []int
	excludeCodes []int
}

// Retry creates retry middleware with provided options
func Retry(attempts int, initialDelay time.Duration, opts ...RetryOption) RoundTripperHandler {
	return func(next http.RoundTripper) http.RoundTripper {
		r := &RetryMiddleware{
			next:         next,
			attempts:     attempts,
			initialDelay: initialDelay,
			maxDelay:     30 * time.Second,
			backoff:      BackoffExponential,
			jitterFactor: 0.1,
		}

		for _, opt := range opts {
			opt(r)
		}

		if len(r.retryCodes) > 0 && len(r.excludeCodes) > 0 {
			panic("retry: cannot use both RetryOnCodes and RetryExcludeCodes")
		}

		return r
	}
}

// RoundTrip implements http.RoundTripper
func (r *RetryMiddleware) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastResponse *http.Response
	var lastError error

	for attempt := 0; attempt < r.attempts; attempt++ {
		if req.Context().Err() != nil {
			return nil, req.Context().Err()
		}

		if attempt > 0 {
			delay := r.calcDelay(attempt)
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(delay):
			}
		}

		resp, err := r.next.RoundTrip(req)
		if err != nil {
			lastError = err
			lastResponse = resp
			continue
		}

		if !r.shouldRetry(resp) {
			return resp, nil
		}

		lastResponse = resp
	}

	if lastError != nil {
		return lastResponse, fmt.Errorf("retry: transport error after %d attempts: %w", r.attempts, lastError)
	}
	return lastResponse, nil
}

func (r *RetryMiddleware) calcDelay(attempt int) time.Duration {
	if attempt == 0 {
		return 0
	}

	var delay time.Duration
	switch r.backoff {
	case BackoffConstant:
		delay = r.initialDelay
	case BackoffLinear:
		delay = r.initialDelay * time.Duration(attempt)
	case BackoffExponential:
		delay = r.initialDelay * time.Duration(math.Pow(2, float64(attempt-1)))
	}

	if delay > r.maxDelay {
		delay = r.maxDelay
	}

	if r.jitterFactor > 0 {
		jitter := float64(delay) * r.jitterFactor
		delay = time.Duration(float64(delay) + rand.Float64()*jitter - jitter/2) //nolint:gosec // week randomness is acceptable
	}

	return delay
}

func (r *RetryMiddleware) shouldRetry(resp *http.Response) bool {
	if len(r.retryCodes) > 0 {
		for _, code := range r.retryCodes {
			if resp.StatusCode == code {
				return true
			}
		}
		return false
	}

	if len(r.excludeCodes) > 0 {
		for _, code := range r.excludeCodes {
			if resp.StatusCode == code {
				return false
			}
		}
		return true
	}

	return resp.StatusCode >= 500
}

// RetryOption represents option type for retry middleware
type RetryOption func(r *RetryMiddleware)

// RetryMaxDelay sets maximum delay between retries
func RetryMaxDelay(d time.Duration) RetryOption {
	return func(r *RetryMiddleware) {
		r.maxDelay = d
	}
}

// RetryWithBackoff sets backoff strategy
func RetryWithBackoff(t BackoffType) RetryOption {
	return func(r *RetryMiddleware) {
		r.backoff = t
	}
}

// RetryWithJitter sets how much randomness to add to delay (0-1.0)
func RetryWithJitter(f float64) RetryOption {
	return func(r *RetryMiddleware) {
		r.jitterFactor = f
	}
}

// RetryOnCodes sets status codes that should trigger a retry
func RetryOnCodes(codes ...int) RetryOption {
	return func(r *RetryMiddleware) {
		r.retryCodes = codes
	}
}

// RetryExcludeCodes sets status codes that should not trigger a retry
func RetryExcludeCodes(codes ...int) RetryOption {
	return func(r *RetryMiddleware) {
		r.excludeCodes = codes
	}
}
