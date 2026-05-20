package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSSRF_BlocksLoopback(t *testing.T) {
	tr := NewNetHTTP(WithAllowHTTP(true))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:9/x", nil)
	_, err := tr.Do(req)
	if err == nil {
		t.Fatalf("expected block, got nil")
	}
	var blocked ErrBlockedHost
	if !errors.As(err, &blocked) {
		t.Errorf("expected ErrBlockedHost, got %T: %v", err, err)
	}
}

func TestSSRF_BlocksRFC1918(t *testing.T) {
	tr := NewNetHTTP(WithAllowHTTP(true))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://10.0.0.1/x", nil)
	_, err := tr.Do(req)
	var blocked ErrBlockedHost
	if !errors.As(err, &blocked) {
		t.Errorf("expected ErrBlockedHost for 10.0.0.1, got %T: %v", err, err)
	}
}

func TestSSRF_BlocksLinkLocal(t *testing.T) {
	tr := NewNetHTTP(WithAllowHTTP(true))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://169.254.169.254/latest/meta-data/", nil)
	_, err := tr.Do(req)
	var blocked ErrBlockedHost
	if !errors.As(err, &blocked) {
		t.Errorf("expected ErrBlockedHost for IMDS, got %T: %v", err, err)
	}
}

func TestSSRF_AllowedHostBypass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	tr := NewNetHTTP(WithAllowHTTP(true), WithAllowedHosts("127.0.0.1"))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tr.Do(req)
	if err != nil {
		t.Fatalf("expected success with allowlist, got %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

func TestRefusesHTTPByDefault(t *testing.T) {
	tr := NewNetHTTP() // no AllowHTTP
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/", nil)
	_, err := tr.Do(req)
	if err == nil || !strings.Contains(err.Error(), "non-HTTPS") {
		t.Errorf("expected non-HTTPS refusal, got %v", err)
	}
}
