package middleware

import (
	"net/http"
)

// MaxConcurrent middleware limits the total concurrency for a given requester
func MaxConcurrent(max int) func(http.RoundTripper) http.RoundTripper {
	sema := make(chan struct{}, max)
	return func(next http.RoundTripper) http.RoundTripper {
		fn := func(req *http.Request) (*http.Response, error) {
			sema <- struct{}{}
			defer func() {
				<-sema
			}()
			return next.RoundTrip(req)
		}
		return RoundTripperFunc(fn)
	}
}
