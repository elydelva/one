package transport

import (
	"errors"
	"net/http"
	"time"

	"elydelva/one/internal/ports"
)

// Config tunes the NetHTTP transport.
type Config struct {
	Timeout       time.Duration
	RedirectCap   int
	AllowHTTP     bool     // false in prod: only https:// scheme accepted
	AllowedHosts  []string // bypass SSRF guard (test-only escape, e.g. ["127.0.0.1"])
}

// Option mutates Config.
type Option func(*Config)

// WithTimeout sets the request timeout (default 30s).
func WithTimeout(d time.Duration) Option { return func(c *Config) { c.Timeout = d } }

// WithRedirectCap sets the max number of redirects (default 5).
func WithRedirectCap(n int) Option { return func(c *Config) { c.RedirectCap = n } }

// WithAllowHTTP allows http:// URLs (default false). Test/dev only.
func WithAllowHTTP(allow bool) Option { return func(c *Config) { c.AllowHTTP = allow } }

// WithAllowedHosts bypasses the SSRF guard for the given hostnames or IPs.
// Test-only — never use in production code paths.
func WithAllowedHosts(hosts ...string) Option {
	return func(c *Config) { c.AllowedHosts = append(c.AllowedHosts, hosts...) }
}

// NetHTTP wraps the standard net/http.Client with SSRF block + scheme guard.
type NetHTTP struct {
	client    *http.Client
	cfg       Config
}

// NewNetHTTP creates a hardened transport.
func NewNetHTTP(opts ...Option) *NetHTTP {
	cfg := Config{Timeout: 30 * time.Second, RedirectCap: 5}
	for _, opt := range opts {
		opt(&cfg)
	}
	dialer := newGuardingDialer(cfg.AllowedHosts)
	tr := &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &NetHTTP{
		client: &http.Client{
			Timeout:       cfg.Timeout,
			Transport:     tr,
			CheckRedirect: checkRedirect(cfg.RedirectCap, cfg.AllowHTTP),
		},
		cfg: cfg,
	}
}

// NewNetHTTPWithClient is kept for tests that need to wrap a custom client.
// SSRF guard does NOT apply to the wrapped client; use only in test code.
func NewNetHTTPWithClient(client *http.Client) *NetHTTP {
	return &NetHTTP{client: client}
}

func (t *NetHTTP) Do(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "https" && !t.cfg.AllowHTTP && t.client.Transport != nil {
		return nil, errors.New("transport: non-HTTPS URL refused: " + req.URL.String())
	}
	return t.client.Do(req) //nolint:gosec
}

var _ ports.Transport = (*NetHTTP)(nil)
