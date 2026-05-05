package middleware

import (
	"bytes"
	"fmt"
	"io"
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
// For requests with bodies (POST, PUT, PATCH), the middleware handles body replay:
// - If req.GetBody is set (automatic for strings.Reader, bytes.Buffer, bytes.Reader), it uses that
// - If req.GetBody is nil and body buffering is disabled (default), requests won't be retried
// - If body buffering is enabled with RetryBufferBodies(true), bodies up to maxBufferSize are buffered
//
// Default configuration:
//   - 3 attempts
//   - Initial delay: 100ms
//   - Max delay: 30s
//   - Exponential backoff
//   - 10% jitter
//   - Retries on 5xx status codes
//   - Body buffering disabled (preserves streaming, no retries for bodies without GetBody)
type RetryMiddleware struct {
	next          http.RoundTripper
	attempts      int
	initialDelay  time.Duration
	maxDelay      time.Duration
	backoff       BackoffType
	jitterFactor  float64
	retryCodes    []int
	excludeCodes  []int
	bufferBodies  bool
	maxBufferSize int64
}

// Retry creates retry middleware with provided options
func Retry(attempts int, initialDelay time.Duration, opts ...RetryOption) RoundTripperHandler {
	return func(next http.RoundTripper) http.RoundTripper {
		r := &RetryMiddleware{
			next:          next,
			attempts:      attempts,
			initialDelay:  initialDelay,
			maxDelay:      30 * time.Second,
			backoff:       BackoffExponential,
			jitterFactor:  0.1,
			bufferBodies:  false,            // disabled by default to preserve streaming; when enabled, reads entire body into memory
			maxBufferSize: 10 * 1024 * 1024, // 10MB limit when buffering enabled
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
	// determine effective attempts based on body handling
	attempts := r.attempts
	hasBody := req.Body != nil && req.Body != http.NoBody

	// prepare body for retries if needed
	if hasBody && req.GetBody == nil && r.attempts > 1 {
		if r.bufferBodies {
			// try to buffer body for retries
			if err := r.bufferRequestBody(req); err != nil {
				// buffering failed or body too large
				return nil, err
			}
		} else {
			// buffering disabled - can't retry with body
			attempts = 1
		}
	}

	var lastResponse *http.Response
	var lastError error

	for attempt := 0; attempt < attempts; attempt++ {
		if req.Context().Err() != nil {
			return nil, fmt.Errorf("retry: context error: %w", req.Context().Err())
		}

		if attempt > 0 {
			delay := r.calcDelay(attempt)
			select {
			case <-req.Context().Done():
				return nil, fmt.Errorf("retry: context cancelled during delay: %w", req.Context().Err())
			case <-time.After(delay):
			}

			// reset body for retry
			if req.GetBody != nil {
				newBody, err := req.GetBody()
				if err != nil {
					return nil, fmt.Errorf("retry: failed to get new request body: %w", err)
				}
				req.Body = newBody
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
		return lastResponse, fmt.Errorf("retry: transport error after %d attempts: %w", attempts, lastError)
	}
	return lastResponse, nil
}

// bufferRequestBody attempts to buffer the request body for retries
// this consumes the original body - returns error if body is too large
func (r *RetryMiddleware) bufferRequestBody(req *http.Request) error {
	// read entire body (with limit for safety)
	bodyBytes, err := io.ReadAll(io.LimitReader(req.Body, r.maxBufferSize+1))
	if err != nil {
		return fmt.Errorf("retry: failed to read request body: %w", err)
	}
	_ = req.Body.Close()

	// check if body exceeds limit
	if int64(len(bodyBytes)) > r.maxBufferSize {
		return fmt.Errorf("retry: request body too large (%d bytes exceeds %d byte limit) - cannot retry",
			len(bodyBytes), r.maxBufferSize)
	}

	// set up body and GetBody for retries
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bodyBytes)), nil
	}

	return nil
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

// RetryBufferBodies enables or disables automatic body buffering for retries
func RetryBufferBodies(enabled bool) RetryOption {
	return func(r *RetryMiddleware) {
		r.bufferBodies = enabled
	}
}

// RetryMaxBufferSize sets the maximum size of request bodies that will be buffered
func RetryMaxBufferSize(size int64) RetryOption {
	return func(r *RetryMiddleware) {
		r.maxBufferSize = size
	}
}
