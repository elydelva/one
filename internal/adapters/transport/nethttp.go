package transport

import (
	"net/http"
	"time"

	"elydelva/one/internal/ports"
)

// NetHTTP wraps the standard net/http.Client.
type NetHTTP struct{ client *http.Client }

// NewNetHTTP creates a transport with a default 30s timeout.
func NewNetHTTP() *NetHTTP {
	return &NetHTTP{client: &http.Client{Timeout: 30 * time.Second}}
}

// NewNetHTTPWithClient creates a transport wrapping the given client.
func NewNetHTTPWithClient(client *http.Client) *NetHTTP {
	return &NetHTTP{client: client}
}

func (t *NetHTTP) Do(req *http.Request) (*http.Response, error) {
	return t.client.Do(req) //nolint:gosec
}

var _ ports.Transport = (*NetHTTP)(nil)
