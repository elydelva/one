package ports

import "net/http"

// Transport executes HTTP requests. Wraps net/http.Client for testability.
type Transport interface {
	Do(req *http.Request) (*http.Response, error)
}
