# Requester

[![Build Status](https://github.com/go-pkgz/requester/workflows/build/badge.svg)](https://github.com/go-pkgz/requester/actions) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/requester/badge.svg?branch=main)](https://coveralls.io/github/go-pkgz/requester?branch=main) [![Go Reference](https://pkg.go.dev/badge/github.com/go-pkgz/requester.svg)](https://pkg.go.dev/github.com/go-pkgz/requester)


The package provides a very thin wrapper (no external dependencies) for `http.Client`, allowing the use of layers (middlewares) at the `http.RoundTripper` level. The goal is to maintain the way users leverage the stdlib HTTP client while adding a few useful extras on top of the standard `http.Client`.

_Please note: this is not a replacement for `http.Client`, but rather a companion library._

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

## Overview

*Built-in middlewares:*

- `Header` - appends user-defined headers to all requests. 
- `MaxConcurrent` - sets maximum concurrency
- `Retry` - sets retry on errors and status codes
- `JSON` - sets headers `"Content-Type": "application/json"` and `"Accept": "application/json"`
- `BasicAuth(user, passwd string)` - adds HTTP Basic Authentication

*Interfaces for external middlewares:*

- `Repeater` - sets repeater to retry failed requests. Doesn't provide repeater implementation but wraps it. Compatible with any repeater (for example [go-pkgz/repeater](https://github.com/go-pkgz/repeater)) implementing a single method interface `Do(ctx context.Context, fun func() error, errors ...error) (err error)` interface. 
- `Cache` - sets any `LoadingCache` implementation to be used for request/response caching. Doesn't provide cache, but wraps it. Compatible with any cache (for example a family of caches from [go-pkgz/lcw](https://github.com/go-pkgz/lcw)) implementing a single-method interface `Get(key string, fn func() (interface{}, error)) (val interface{}, err error)`
- `Logger` - sets logger, compatible with any implementation  of a single-method interface `Logf(format string, args ...interface{})`, for example [go-pkgz/lgr](https://github.com/go-pkgz/lgr)
- `CircuitBreaker` - sets circuit breaker, interface compatible with [sony/gobreaker](https://github.com/sony/gobreaker)

Users can add any custom middleware. All it needs is a handler `RoundTripperHandler func(http.RoundTripper) http.RoundTripper`. 
Convenient functional adapter `middleware.RoundTripperFunc` provided.
 
See examples of the usage in [_example](https://github.com/go-pkgz/requester/tree/master/_example)

### Header middleware

`Header` middleware adds user-defined headers to all requests. It expects a map of headers to be added. For example:

```go
rq := requester.New(http.Client{}, middleware.Header("X-Auth", "123456789"))
```
### MaxConcurrent middleware

`MaxConcurrent` middleware can be used to limit the concurrency of a given requester and limit overall concurrency for multiple requesters. For the first case, `MaxConcurrent(N)` should be created in the requester chain of middlewares. For example, `rq := requester.New(http.Client{Timeout: 3 * time.Second}, middleware.MaxConcurrent(8))`. To make it global, `MaxConcurrent` should be created once, outside the chain, and passed into each requester. For example:

```go
mc := middleware.MaxConcurrent(16)
rq1 := requester.New(http.Client{Timeout: 3 * time.Second}, mc)
rq2 := requester.New(http.Client{Timeout: 1 * time.Second}, middleware.JSON, mc)
```
### Retry middleware

Retry middleware provides a flexible retry mechanism with different backoff strategies. By default, it retries on network errors and 5xx responses.

```go
// retry 3 times with exponential backoff, starting from 100ms
rq := requester.New(http.Client{}, middleware.Retry(3, 100*time.Millisecond))

// retry with custom options
rq := requester.New(http.Client{}, middleware.Retry(3, 100*time.Millisecond,
    middleware.RetryWithBackoff(middleware.BackoffLinear),    // use linear backoff
    middleware.RetryMaxDelay(5*time.Second),                  // cap maximum delay
    middleware.RetryWithJitter(0.1),                          // add 10% randomization
    middleware.RetryOnCodes(503, 502),                        // retry only on specific codes
    // or middleware.RetryExcludeCodes(404, 401),             // alternatively, retry on all except these codes
))
```

Default configuration:
- 3 attempts
- Initial delay: 100ms
- Max delay: 30s
- Exponential backoff
- 10% jitter
- Retries on 5xx status codes

Retry Options:
- `RetryWithBackoff(t BackoffType)` - set backoff strategy (Constant, Linear, or Exponential)
- `RetryMaxDelay(d time.Duration)` - cap the maximum delay between retries
- `RetryWithJitter(f float64)` - add randomization to delays (0-1.0 factor)
- `RetryOnCodes(codes ...int)` - retry only on specific status codes
- `RetryExcludeCodes(codes ...int)` - retry on all codes except specified

Note: `RetryOnCodes` and `RetryExcludeCodes` are mutually exclusive and can't be used together.

### Cache middleware

Cache middleware provides an **in-memory caching layer** for HTTP responses. It improves performance by avoiding repeated network calls for the same request.

#### **Basic Usage**

```go
rq := requester.New(http.Client{}, middleware.Cache())
```

By default:

- Only GET requests are cached
- TTL (Time-To-Live) is 5 minutes
- Maximum cache size is 1000 entries
- Caches only HTTP 200 responses


#### **Cache Configuration Options**

```go
rq := requester.New(http.Client{}, middleware.Cache(
    middleware.CacheTTL(10*time.Minute),        // change TTL to 10 minutes
    middleware.CacheSize(500),                  // limit cache to 500 entries
    middleware.CacheMethods(http.MethodGet, http.MethodPost), // allow caching for GET and POST
    middleware.CacheStatuses(200, 201, 204),    // cache only responses with these status codes
    middleware.CacheWithBody,                   // include request body in cache key
    middleware.CacheWithHeaders("Authorization", "X-Custom-Header"), // include selected headers in cache key
))
```

#### Cache Key Composition

By default, the cache key is generated using:

- HTTP **method**
- Full **URL**
- (Optional) **Headers** (if `CacheWithHeaders` is enabled)
- (Optional) **Body** (if `CacheWithBody` is enabled)

For example, enabling `CacheWithHeaders("Authorization")` will cache the same URL differently **for each unique Authorization token**.

#### Cache Eviction Strategy

- **Entries expire** when the TTL is reached.
- **If the cache reaches its maximum size**, the **oldest entry is evicted** (FIFO order).


#### Cache Limitations

- **Only caches complete HTTP responses.** Streaming responses are **not** supported.
- **Does not cache responses with status codes other than 200** (unless explicitly allowed).
- **Uses in-memory storage**, meaning the cache **resets on application restart**.


### JSON middleware

`JSON` middleware sets headers `"Content-Type": "application/json"` and `"Accept": "application/json"`.

```go
rq := requester.New(http.Client{}, middleware.JSON)
```
    
### BasicAuth middleware

`BasicAuth` middleware adds HTTP Basic Authentication to all requests. It expects a username and password. For example:

```go
rq := requester.New(http.Client{}, middleware.BasicAuth("user", "passwd"))
```

----

### Logging middleware interface

Logger should implement `Logger` interface with a single method `Logf(format string, args ...interface{})`. 
For convenience, func type `LoggerFunc` is provided as an adapter to allow the use of ordinary functions as `Logger`. 

Two basic implementations included: 

- `NoOpLogger` do-nothing logger (default) 
- `StdLogger` wrapper for stdlib logger.

logging options:

- `Prefix(prefix string)` sets prefix for each logged line
- `WithBody` - allows request's body logging
- `WithHeaders` - allows request's headers logging

Note: If logging is allowed, it will log the URL, method, and may log headers and the request body. It may affect application security. For example, if a request passes some sensitive information as part of the body or header. In this case, consider turning logging off or providing your own logger to suppress all that you need to hide.


If the request is limited, it will wait till the limit is released.

### Cache middleware interface

Cache expects the `LoadingCache` interface to implement a single method: `Get(key string, fn func() (interface{}, error)) (val interface{}, err error)`. [LCW](https://github.com/go-pkgz/lcw/) can be used directly, and in order to adopt other caches, see the provided `LoadingCacheFunc`.

#### Caching Key and Allowed Requests

By default, only `GET` calls are cached. This can be changed with the `Methods(methods ...string)` option. The default key is composed of the full URL.

Several options define what part of the request will be used for the key:

-  `KeyWithHeaders` - adds all headers to a key
-  `KeyWithHeadersIncluded(headers ...string)` - adds only requested headers
-  `KeyWithHeadersExcluded(headers ...string)` - adds all headers excluded
-  `KeyWithBody` - adds the request's body, limited to the first 16k of the body
-  `KeyFunc` - any custom logic provided by the caller

example: `cache.New(lruCache, cache.Methods("GET", "POST"), cache.KeyFunc() {func(r *http.Request) string {return r.Host})`

#### cache and streaming response

`Cache` is **not compatible** with HTTP streaming mode. Practically, this is rare and exotic, but allowing `Cache` will effectively transform the streaming response into a "get it all" typical response. This is due to the fact that the cache has to read the response body fully to save it, so technically streaming will be working, but the client will receive all the data at once.

### Repeater middleware interface

`Repeater` expects a single method interface `Do(fn func() error, failOnCodes ...error) (err error)`. [repeater](github.com/go-pkgz/repeater) can be used directly.

By default, the repeater will retry on any error and any status code >= 400. However, the user can pass `failOnCodes` to explicitly define which status codes should be treated as errors and retry only on those. For example: `Repeater(repeaterSvc, 500, 400)` repeats requests on 500 and 400 statuses only.

In a special case where the user wants to retry only on the underlying transport errors (network, timeouts, etc.) and not on any status codes `Repeater(repeaterSvc, 0)` can be used.

### User-Defined Middlewares

Users can add any additional handlers (middleware) to the chain. Each middleware provides `middleware.RoundTripperHandler` and
can alter the request or implement any other custom functionality.

Example of a handler resetting a particular header:

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

-  Pass it to the `New` constructor, i.e. `requester.New(http.Client{}, middleware.MaxConcurrent(8), middleware.Header("foo", "bar"))`
-  Add after construction with the `Use` method
-  Create a new, inherited requester by using `With`:

```go
rq := requester.New(http.Client{}, middleware.Header("foo", "bar")) // make requester enforcing header foo:bar
resp, err := rq.Do(some_http_req) // send a request

rqLimited := rq.With(middleware.MaxConcurrent(8)) // make requester from rq (foo:bar enforced) and add 8 max concurrency
resp, err := rqLimited.Do(some_http_req)
```

## Getting http.Client with all middlewares

For convenience, `requester.Client()` returns `*http.Client` with all middlewares injected. From this point, the user can call `Do` on this client, and it will invoke the request with all the middlewares.

## Helpers and adapters

- `CircuitBreakerFunc func(req func() (interface{}, error)) (interface{}, error)` - adapter to allow the use of an ordinary functions as CircuitBreakerSvc.
- `logger.Func func(format string, args ...interface{})` - functional adapter for `logger.Service`.
- `cache.ServiceFunc func(key string, fn func() (interface{}, error)) (interface{}, error)` - functional adapter for `cache.Service`.
- `RoundTripperFunc func(*http.Request) (*http.Response, error)` - functional adapter for RoundTripperHandler
