package runtime

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"elydelva/one/internal/core"
)

func resp(status int, body string, headers http.Header) *http.Response {
	if headers == nil {
		headers = http.Header{}
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     headers,
	}
}

func TestMapHTTPError_401(t *testing.T) {
	r := resp(401, "bad", nil)
	defer func() { _ = r.Body.Close() }()
	err := MapHTTPError(r, core.Action{Service: "github", ID: "issues.read"})
	var notAuth core.ErrNotAuthenticated
	if !errors.As(err, &notAuth) {
		t.Errorf("got %T: %v", err, err)
	}
}

func TestMapHTTPError_403(t *testing.T) {
	r := resp(403, "no", nil)
	defer func() { _ = r.Body.Close() }()
	err := MapHTTPError(r, core.Action{Service: "github", ID: "issues.read"})
	var f core.ErrForbidden
	if !errors.As(err, &f) {
		t.Errorf("got %T: %v", err, err)
	}
}

func TestMapHTTPError_404WithHint(t *testing.T) {
	r := resp(404, "", nil)
	defer func() { _ = r.Body.Close() }()
	act := core.Action{
		Service: "notion", ID: "pages.read",
		Errors: map[int]core.ErrorSpec{404: {Code: "not_found", Hint: "Share the page with the integration.", InstallGuide: "share-page"}},
	}
	err := MapHTTPError(r, act)
	var nf core.ErrNotFound
	if !errors.As(err, &nf) {
		t.Fatalf("got %T: %v", err, err)
	}
	if nf.Hint == "" || nf.Guide != "share-page" {
		t.Errorf("hint/guide not propagated: %+v", nf)
	}
}

func TestMapHTTPError_429(t *testing.T) {
	h := http.Header{"Retry-After": []string{"60"}}
	r := resp(429, "", h)
	defer func() { _ = r.Body.Close() }()
	err := MapHTTPError(r, core.Action{Service: "github"})
	var rl core.ErrRateLimited
	if !errors.As(err, &rl) {
		t.Fatalf("got %T", err)
	}
	if rl.RetryAfter != "60" {
		t.Errorf("retry-after = %q", rl.RetryAfter)
	}
}

func TestMapHTTPError_5xx(t *testing.T) {
	r := resp(503, "down", nil)
	defer func() { _ = r.Body.Close() }()
	err := MapHTTPError(r, core.Action{Service: "github"})
	var api core.ErrAPIError
	if !errors.As(err, &api) || api.Status != 503 {
		t.Errorf("got %T: %v", err, err)
	}
}
