package middleware

import (
	"net/http"
)

// Header middleware adds a header to request
func Header(key, value string) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		fn := func(req *http.Request) (*http.Response, error) {
			req.Header.Set(key, value)
			return next.RoundTrip(req)
		}
		return RoundTripperFunc(fn)
	}
}

// JSON sets Content-Type and Accept headers to json
func JSON(next http.RoundTripper) http.RoundTripper {
	fn := func(req *http.Request) (*http.Response, error) {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		return next.RoundTrip(req)
	}
	return RoundTripperFunc(fn)
}

// BasicAuth middleware adds basic auth to request
func BasicAuth(user, passwd string) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		fn := func(req *http.Request) (*http.Response, error) {
			req.SetBasicAuth(user, passwd)
			return next.RoundTrip(req)
		}
		return RoundTripperFunc(fn)
	}
}
