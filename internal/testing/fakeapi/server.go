// Package fakeapi provides an httptest helper for E2E and adapter tests.
//
// Routes are declared via a small table; each request is matched on method+path
// (path supports `:param` placeholders). The handler returns the configured
// status + JSON body. All requests are recorded for assertions.
package fakeapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Route declares a canned response for an incoming request.
type Route struct {
	Method string
	Path   string // supports `:name` placeholders (e.g. /repos/:owner/:repo)
	Status int
	Body   any // marshaled as JSON; if string, sent as-is
	// Optional: callback that runs on match; can inspect the request.
	OnMatch func(r *http.Request)
}

// Recorded captures one received request for assertion.
type Recorded struct {
	Method string
	Path   string
	Query  map[string]string
	Header http.Header
	Body   []byte
}

// Server wraps httptest.Server with route table + request recording.
type Server struct {
	*httptest.Server
	mu       sync.Mutex
	routes   []Route
	received []Recorded
}

// New starts a fakeapi.Server with the given routes.
func New(t *testing.T, routes []Route) *Server {
	t.Helper()
	s := &Server{routes: routes}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.Server.Close)
	return s
}

// Received returns the requests received so far.
func (s *Server) Received() []Recorded {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Recorded, len(s.received))
	copy(out, s.received)
	return out
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	rec := Recorded{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  map[string]string{},
		Header: r.Header.Clone(),
		Body:   body,
	}
	for k := range r.URL.Query() {
		rec.Query[k] = r.URL.Query().Get(k)
	}
	s.mu.Lock()
	s.received = append(s.received, rec)
	s.mu.Unlock()

	for _, route := range s.routes {
		if route.Method != r.Method {
			continue
		}
		if !pathMatches(route.Path, r.URL.Path) {
			continue
		}
		if route.OnMatch != nil {
			route.OnMatch(r)
		}
		status := route.Status
		if status == 0 {
			status = 200
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		switch b := route.Body.(type) {
		case nil:
			// no body
		case string:
			_, _ = io.WriteString(w, b)
		case []byte:
			_, _ = w.Write(b)
		default:
			_ = json.NewEncoder(w).Encode(b)
		}
		return
	}
	w.WriteHeader(http.StatusNotFound)
	_, _ = io.WriteString(w, `{"error":"fakeapi: no matching route"}`)
}

// pathMatches returns true when `pattern` (with `:name` placeholders) matches `actual`.
func pathMatches(pattern, actual string) bool {
	pp := strings.Split(strings.Trim(pattern, "/"), "/")
	ap := strings.Split(strings.Trim(actual, "/"), "/")
	if len(pp) != len(ap) {
		return false
	}
	for i := range pp {
		if strings.HasPrefix(pp[i], ":") {
			continue
		}
		if pp[i] != ap[i] {
			return false
		}
	}
	return true
}
