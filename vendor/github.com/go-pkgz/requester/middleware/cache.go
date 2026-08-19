package middleware

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// CacheEntry represents a cached response with metadata
type CacheEntry struct {
	body      []byte
	headers   http.Header
	status    int
	createdAt time.Time
}

// CacheMiddleware implements in-memory cache for HTTP responses with TTL-based eviction
type CacheMiddleware struct {
	next           http.RoundTripper
	ttl            time.Duration
	maxKeys        int
	includeBody    bool
	headers        []string
	allowedCodes   []int
	allowedMethods []string

	cache map[string]CacheEntry
	keys  []string // maintains insertion order
	mu    sync.Mutex
}

// RoundTrip implements http.RoundTripper
func (c *CacheMiddleware) RoundTrip(req *http.Request) (*http.Response, error) {
	// check if method is allowed
	methodAllowed := slices.Contains(c.allowedMethods, req.Method)
	if !methodAllowed {
		resp, err := c.next.RoundTrip(req)
		if err != nil {
			return resp, fmt.Errorf("cache: transport error: %w", err)
		}
		return resp, nil
	}

	key := c.makeKey(req) // generate cache key based on request

	c.mu.Lock()
	// remove expired entries
	for len(c.keys) > 0 {
		oldestKey := c.keys[0]
		if time.Since(c.cache[oldestKey].createdAt) < c.ttl {
			break // stop once we find a non-expired entry
		}
		delete(c.cache, oldestKey)
		c.keys = c.keys[1:]
	}
	// check cache
	entry, found := c.cache[key]
	c.mu.Unlock()

	if found {
		// cache hit - reconstruct response
		return &http.Response{
			Status:        fmt.Sprintf("%d %s", entry.status, http.StatusText(entry.status)),
			StatusCode:    entry.status,
			Header:        entry.headers.Clone(),
			Body:          io.NopCloser(bytes.NewReader(entry.body)),
			Request:       req,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: int64(len(entry.body)),
		}, nil
	}

	// fetch fresh response
	resp, err := c.next.RoundTrip(req)
	if err != nil {
		return resp, fmt.Errorf("cache: transport error: %w", err)
	}

	// check if response code is allowed for caching
	if !c.shouldCache(resp.StatusCode) {
		return resp, nil
	}

	// read and store response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, fmt.Errorf("cache: failed to read response body: %w", err)
	}
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	// store in cache
	c.mu.Lock()
	defer c.mu.Unlock()

	// evict oldest if maxKeys reached
	if len(c.cache) >= c.maxKeys {
		oldestKey := c.keys[0]
		delete(c.cache, oldestKey)
		c.keys = c.keys[1:]
	}

	// store new entry
	c.cache[key] = CacheEntry{body: body, headers: resp.Header.Clone(), status: resp.StatusCode, createdAt: time.Now()}
	c.keys = append(c.keys, key) // maintain order of keys for LRU eviction

	return resp, nil
}

// makeKey generates a cache key based on the request details.
// every part goes in as a length-prefixed field, so a value of one part can't be read as another one
func (c *CacheMiddleware) makeKey(req *http.Request) string {
	// req.Host overrides the URL host on the wire, responses from different virtual hosts must not share an entry
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}

	body := ""
	if c.includeBody && req.Body != nil {
		if b, err := io.ReadAll(req.Body); err == nil {
			body = string(b)
			req.Body = io.NopCloser(bytes.NewReader(b))
		}
	}

	headersKey := ""
	if len(c.headers) > 0 {
		var headers []string
		for _, h := range c.headers {
			vals := req.Header.Values(h)
			if len(vals) == 0 {
				continue
			}
			var hb strings.Builder
			fmt.Fprintf(&hb, "%d:%s", len(h), h)
			for _, v := range vals {
				fmt.Fprintf(&hb, "%d:%s", len(v), v)
			}
			// the whole record is length-prefixed too, a multi-valued header can't look like several headers
			headers = append(headers, fmt.Sprintf("%d:%s", hb.Len(), hb.String()))
		}
		sort.Strings(headers)
		headersKey = strings.Join(headers, "")
	}

	var sb strings.Builder
	for _, part := range []string{req.Method, req.URL.String(), host, body, headersKey} {
		fmt.Fprintf(&sb, "%d:%s", len(part), part)
	}

	hash := sha256.Sum256([]byte(sb.String()))
	return fmt.Sprintf("%x", hash)
}

func (c *CacheMiddleware) shouldCache(code int) bool {
	return slices.Contains(c.allowedCodes, code)
}

// Cache creates caching middleware with provided options
func Cache(opts ...CacheOption) RoundTripperHandler {
	return func(next http.RoundTripper) http.RoundTripper {
		c := &CacheMiddleware{
			next:           next,
			ttl:            5 * time.Minute,
			maxKeys:        1000,
			allowedCodes:   []int{200},
			allowedMethods: []string{http.MethodGet},
			cache:          make(map[string]CacheEntry),
			keys:           make([]string, 0, 1000),
		}

		for _, opt := range opts {
			opt(c)
		}

		return c
	}
}

// CacheOption represents cache middleware options
type CacheOption func(c *CacheMiddleware)

// CacheTTL sets cache TTL
func CacheTTL(ttl time.Duration) CacheOption {
	return func(c *CacheMiddleware) {
		c.ttl = ttl
	}
}

// CacheSize sets maximum number of cached entries
func CacheSize(size int) CacheOption {
	return func(c *CacheMiddleware) {
		c.maxKeys = size
	}
}

// CacheWithBody includes request body in cache key
func CacheWithBody(c *CacheMiddleware) {
	c.includeBody = true
}

// CacheWithHeaders includes specified headers in cache key
func CacheWithHeaders(headers ...string) CacheOption {
	return func(c *CacheMiddleware) {
		c.headers = headers
	}
}

// CacheStatuses sets which response status codes should be cached
func CacheStatuses(codes ...int) CacheOption {
	return func(c *CacheMiddleware) {
		c.allowedCodes = codes
	}
}

// CacheMethods sets which HTTP methods should be cached
func CacheMethods(methods ...string) CacheOption {
	return func(c *CacheMiddleware) {
		c.allowedMethods = methods
	}
}
