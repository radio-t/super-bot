# Requester

[![Build Status](https://github.com/go-pkgz/requester/workflows/build/badge.svg)](https://github.com/go-pkgz/requester/actions) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/requester/badge.svg?branch=main)](https://coveralls.io/github/go-pkgz/requester?branch=main) [![Go Reference](https://pkg.go.dev/badge/github.com/go-pkgz/requester.svg)](https://pkg.go.dev/github.com/go-pkgz/requester)


The package provides a very thin wrapper (no external dependencies) for `http.Client`, allowing to use of layers (middlewares) on `http.RoundTripper` level. The goal is to keep the way users leverage stdlib http client but add a few useful extras on top of the standard `http.Client`.

_Pls note: this is not a replacement for http.Client but rather a companion library._

```go
    rq := requester.New(                        // make the requester
        http.Client{Timeout: 5*time.Second},    // set http client
        requester.MaxConcurrent(8),             // maximum number of concurrent requests
        requester.JSON,                         // set json headers
        requester.Header("X-AUTH", "123456789"),// set some auth header
        requester.Logger(requester.StdLogger),  // enable logging to stdout
    )
    
    req := http.NewRequest("GET", "http://example.com/api", nil) // create the usual http.Request
    req.Header.Set("foo", "bar") // do the usual things with request, for example set some custome headers
    resp, err := rq.Do(req) // instead of client.Do call requester.Do
```


## Install and update

`go get -u github.com/go-pkgz/requester`


## `Requester` middlewares 

## Overview

- `Header` - appends user-defined headers to all requests. 
- `JSON` - sets headers `"Content-Type": "application/json"` and `"Accept": "application/json"`
- `BasicAuth(user, passwd string)` - adds HTTP Basic Authentication
- `MaxConcurrent` - sets maximum concurrency
- `Repeater` - sets repeater to retry failed requests. Doesn't provide repeater implementation but wraps it. Compatible with any repeater (for example [go-pkgz/repeater](https://github.com/go-pkgz/repeater)) implementing a single method interface `Do(ctx context.Context, fun func() error, errors ...error) (err error)` interface. 
- `Cache` - sets any `LoadingCache` implementation to be used for request/response caching. Doesn't provide cache, but wraps it. Compatible with any cache (for example a family of caches from [go-pkgz/lcw](https://github.com/go-pkgz/lcw)) implementing a single-method interface `Get(key string, fn func() (interface{}, error)) (val interface{}, err error)`
- `Logger` - sets logger, compatible with any implementation  of a single-method interface `Logf(format string, args ...interface{})`, for example [go-pkgz/lgr](https://github.com/go-pkgz/lgr)
- `CircuitBreaker` - sets circuit breaker, interface compatible with [sony/gobreaker](https://github.com/sony/gobreaker)

Users can add any custom middleware. All it needs is a handler `RoundTripperHandler func(http.RoundTripper) http.RoundTripper`. 
Convenient functional adapter `middleware.RoundTripperFunc` provided.
 
See examples of the usage in [_example](https://github.com/go-pkgz/requester/tree/master/_example)

### Logging 

Logger should implement `Logger` interface with a single method `Logf(format string, args ...interface{})`. 
For convenience, func type `LoggerFunc` is provided as an adapter to allow the use of ordinary functions as `Logger`. 

Two basic implementation included: 

- `NoOpLogger` do-nothing logger (default) 
- `StdLogger` wrapper for stdlib logger.

logging options:

- `Prefix(prefix string)` sets prefix for each logged line
- `WithBody` - allows request's body logging
- `WithHeaders` - allows request's headers logging

Note: if logging is allowed, it will log URL, method, and may log headers and the request body. 
It may affect application security. For example, if a request passes some sensitive info as a part of the body or header. 
In this case, consider turning logging off or provide your own logger suppressing all you need to hide. 

### MaxConcurrent

MaxConcurrent middleware can be used to limit the concurrency of a given requester and limit overall concurrency for multiple
requesters. For the first case, `MaxConcurrent(N)` should be created in the requester chain of middlewares, for example, `rq := requester.New(http.Client{Timeout: 3 * time.Second}, middleware.MaxConcurrent(8))`. To make it global, `MaxConcurrent` should be created once, outside the chain and passed into each requester, i.e.

```go
mc := middleware.MaxConcurrent(16)
rq1 := requester.New(http.Client{Timeout: 3 * time.Second}, mc)
rq2 := requester.New(http.Client{Timeout: 1 * time.Second}, middleware.JSON, mc)
```

If request was limited, it will wait till the limit is released.

### Cache

Cache expects `LoadingCache` interface implementing a single method:
`Get(key string, fn func() (interface{}, error)) (val interface{}, err error)`. [LCW](https://github.com/go-pkgz/lcw/) can 
be used directly, and in order to adopt other caches see provided `LoadingCacheFunc`.

#### caching key and allowed requests

By default, only `GET` calls are cached. This can be changed with `Methods(methods ...string)` option.
The default key composed of the full URL.

Several options are defining what part of the request will be used for the key:

- `KeyWithHeaders` - adds all headers to a key
- `KeyWithHeadersIncluded(headers ...string)` - adds only requested headers
- `KeyWithHeadersExcluded(headers ...string) ` - adds all headers excluded
- `KeyWithBody` - adds request's body, limited to the first 16k of the body
- `KeyFunc` - any custom logic provided by the caller

example: `cache.New(lruCache, cache.Methods("GET", "POST"), cache.KeyFunc() {func(r *http.Request) string {return r.Host})`


#### cache and streaming response

`Cache` is **not compatible** with http streaming mode. Practically, this is rare and exotic, but allowing `Cache` will effectively transform the streaming response to a "get all" typical response. It is due to cache
has to read response body fully to save it, so technically streaming will be working, but the client will get
all the data at once. 

### Repeater

`Repeater` expects a single method interface `Do(fn func() error, failOnCodes ...error) (err error)`. [repeater](github.com/go-pkgz/repeater) can be used directly.

By default, the repeater will retry on any error and any status code >=400.
However, user can pass `failOnCodes` to explicitly define what status code should be treated as errors and retry only on those.
For example: `Repeater(repeaterSvc, 500, 400)` repeats requests on 500 and 400 statuses only.

For a special case if user want to retry only on the underlying transport errors (network, timeouts, etc) and not on any status codes,
`Repeater(repeaterSvc, 0)` can be used.

### User-Defined Middlewares

Users can add any additional handlers (middleware) to the chain. Each middleware provides `middleware.RoundTripperHandler` and
can alter the request or implement any other custom functionality.

Example of a handler resetting particular header:

```go
maskHeader := func(http.RoundTripper) http.RoundTripper {
    fn := func(req *http.Request) (*http.Response, error) {
        req.Header.Del("deleteme")
        return next(req)
    }
    return middleware.RoundTripperFunc(fn)
}

rq := requester.New(http.Client{}, maskHeader)
```

## Adding middleware to requester

There are 3 ways to add middleware(s):

- Pass it to `New` constructor, i.e. `requester.New(http.Client{}, middleware.MaxConcurrent(8), middleware.Header("foo", "bar"))`
- Add after construction with `Use` method
- Create new, inherited requester by using `With`:
```go
rq := requester.New(http.Client{}, middleware.Header("foo", "bar")) // make requester enforcing header foo:bar
resp, err := rq.Do(some_http_req) // send a request

rqLimited := rq.With(middleware.MaxConcurrent(8)) // make requester from rq (foo:bar enforced) and add 8 max concurrency
resp, err := rqLimited.Do(some_http_req)
```

## Getting http.Client with all middlewares

For convenience `requester.Client()` returns `*http.Client` with all middlewares injected in. From this point user can call `Do` of this client, and it will invoke the request with all the middlewares.

## Helpers and adapters

- `CircuitBreakerFunc func(req func() (interface{}, error)) (interface{}, error)` - adapter to allow the use of an ordinary functions as CircuitBreakerSvc.
- `logger.Func func(format string, args ...interface{})` - functional adapter for `logger.Service`.
- `cache.ServiceFunc func(key string, fn func() (interface{}, error)) (interface{}, error)` - functional adapter for `cache.Service`.
- `RoundTripperFunc func(*http.Request) (*http.Response, error)` - functional adapter for RoundTripperHandler
