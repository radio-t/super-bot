package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// RepeaterSvc defines repeater interface
type RepeaterSvc interface {
	Do(ctx context.Context, fun func() error, errs ...error) (err error)
}

// Repeater sets middleware with provided RepeaterSvc to retry failed requests
func Repeater(repeater RepeaterSvc, failOnCodes ...int) RoundTripperHandler {

	return func(next http.RoundTripper) http.RoundTripper {

		fn := func(req *http.Request) (*http.Response, error) {
			if repeater == nil {
				return next.RoundTrip(req)
			}

			var resp *http.Response
			var err error
			e := repeater.Do(req.Context(), func() error {
				resp, err = next.RoundTrip(req)
				if err != nil {
					return err
				}
				for _, fc := range failOnCodes {
					if resp.StatusCode == fc {
						return errors.New(resp.Status)
					}
				}
				return nil
			})
			if e != nil {
				return nil, fmt.Errorf("repeater: %w", e)
			}
			return resp, nil
		}
		return RoundTripperFunc(fn)
	}
}
