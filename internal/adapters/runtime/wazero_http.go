//go:build !nowasm

package runtime

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// hostHTTPRequest is the JSON shape passed from a handler to host.http.request.
type hostHTTPRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	// BodyBase64 lets handlers send binary payloads safely through JSON.
	BodyBase64 string `json:"body_b64,omitempty"`
}

// hostHTTPResponse is the JSON shape returned to the handler.
type hostHTTPResponse struct {
	Status     int               `json:"status"`
	Headers    map[string]string `json:"headers,omitempty"`
	BodyBase64 string            `json:"body_b64,omitempty"`
}

// allowlist caches compiled patterns per action handler reference.
var (
	allowlistCache   sync.Map // key: handler sha256 → []*regexp.Regexp
	maxPatternLength = 512
)

func compileAllowlist(h *core.HandlerRef) ([]*regexp.Regexp, error) {
	if v, ok := allowlistCache.Load(h.Sha256); ok {
		return v.([]*regexp.Regexp), nil
	}
	out := make([]*regexp.Regexp, 0, len(h.Calls))
	for _, p := range h.Calls {
		if len(p) == 0 || len(p) > maxPatternLength {
			return nil, fmt.Errorf("allowlist pattern length out of range: %d", len(p))
		}
		// Anchor to full URL. Disallow blanket ".*" patterns.
		if p == ".*" || p == ".+" || p == "^.*$" {
			return nil, fmt.Errorf("allowlist pattern too permissive: %q", p)
		}
		anchored := p
		if anchored[0] != '^' {
			anchored = "^" + anchored
		}
		if anchored[len(anchored)-1] != '$' {
			anchored = anchored + "$"
		}
		re, err := regexp.Compile(anchored)
		if err != nil {
			return nil, fmt.Errorf("allowlist pattern %q: %w", p, err)
		}
		out = append(out, re)
	}
	if h.Sha256 != "" {
		allowlistCache.Store(h.Sha256, out)
	}
	return out, nil
}

func (r *WazeroRuntime) doHostHTTP(ctx context.Context, st *invocationState, raw []byte) ([]byte, error) {
	var hr hostHTTPRequest
	if err := json.Unmarshal(raw, &hr); err != nil {
		st.setViolation("http_bad_request", err.Error())
		return nil, err
	}

	patterns, err := compileAllowlist(st.req.Action.Handler)
	if err != nil {
		st.setViolation("http_allowlist_compile", err.Error())
		return nil, err
	}

	allowed := false
	for _, re := range patterns {
		if re.MatchString(hr.URL) {
			allowed = true
			break
		}
	}
	if !allowed {
		st.setViolation("http_url_not_allowed", hr.URL)
		return nil, fmt.Errorf("url not allowed: %s", hr.URL)
	}

	method := hr.Method
	if method == "" {
		method = "GET"
	}

	var body io.Reader
	if hr.BodyBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(hr.BodyBase64)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(decoded)
	} else if hr.Body != "" {
		body = bytes.NewReader([]byte(hr.Body))
	}

	hreq, err := http.NewRequestWithContext(ctx, method, hr.URL, body)
	if err != nil {
		return nil, err
	}
	for k, v := range hr.Headers {
		hreq.Header.Set(k, v)
	}

	resp, err := r.transport.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	st.httpCalls = append(st.httpCalls, ports.HTTPCall{Method: method, URL: hr.URL, Status: resp.StatusCode})

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	hdr := make(map[string]string, len(resp.Header))
	for k, v := range resp.Header {
		if len(v) > 0 {
			hdr[k] = v[0]
		}
	}
	out := hostHTTPResponse{
		Status:     resp.StatusCode,
		Headers:    hdr,
		BodyBase64: base64.StdEncoding.EncodeToString(bs),
	}
	return json.Marshal(out)
}
