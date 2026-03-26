package httpclient

import (
	"context"
	"io"
	"math"
	"net/http"
	"sync"
	"time"
)

// Client wraps http.Client with timeouts and simple retry.
type Client struct {
	inner *http.Client
}

// New returns a shared-style HTTP client per architecture defaults.
func New() *Client {
	return &Client{
		inner: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxConnsPerHost:     10,
				IdleConnTimeout:     30 * time.Second,
				TLSHandshakeTimeout: 5 * time.Second,
			},
		},
	}
}

// Do executes a request with up to 3 retries on transient errors.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	backoff := time.Second
	for attempt := 0; attempt < 3; attempt++ {
		r := req.Clone(ctx)
		resp, err := c.inner.Do(r)
		if err != nil {
			lastErr = err
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff = time.Duration(math.Min(float64(backoff*2), float64(4*time.Second)))
			continue
		}
		if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lastErr = errTransient(resp.StatusCode)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff = time.Duration(math.Min(float64(backoff*2), float64(4*time.Second)))
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}

func errTransient(code int) error {
	return &statusError{code: code}
}

type statusError struct {
	code int
}

func (e *statusError) Error() string {
	return http.StatusText(e.code)
}

// UserAgentTransport sets User-Agent (required for crates.io; harmless elsewhere).
type UserAgentTransport struct {
	Base      http.RoundTripper
	UserAgent string
}

func (t *UserAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	if r.Header.Get("User-Agent") == "" {
		r.Header.Set("User-Agent", t.UserAgent)
	}
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(r)
}

// PerHostLimiter is a token-bucket style limiter per host (coarse).
type PerHostLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     map[string]time.Time
}

// NewPerHostLimiter returns a limiter with minimum interval between requests per host.
func NewPerHostLimiter(rps float64) *PerHostLimiter {
	interval := time.Second
	if rps > 0 {
		interval = time.Duration(float64(time.Second) / rps)
	}
	return &PerHostLimiter{interval: interval, last: make(map[string]time.Time)}
}

// Wait blocks until this host may send another request.
func (p *PerHostLimiter) Wait(host string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	if t, ok := p.last[host]; ok {
		next := t.Add(p.interval)
		if d := time.Until(next); d > 0 {
			p.mu.Unlock()
			time.Sleep(d)
			p.mu.Lock()
		}
	}
	p.last[host] = time.Now()
}
