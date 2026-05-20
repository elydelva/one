package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// DeclarativeRuntime executes actions defined purely in YAML (HTTP requests, no WASM).
type DeclarativeRuntime struct {
	transport ports.Transport
	clock     ports.Clock

	// resolver allows the caller to look up the service for an action (for
	// base URL and auth injection rules). Required.
	resolver ServiceResolver
}

// ServiceResolver fetches a service definition given a ServiceID. Provided by
// the composition root (typically wraps a Catalog).
type ServiceResolver interface {
	Resolve(ctx context.Context, id core.ServiceID) (*core.Service, error)
}

// CatalogResolver adapts a ports.Catalog to ServiceResolver.
type CatalogResolver struct{ Catalog ports.Catalog }

// NewCatalogResolver wraps a Catalog.
func NewCatalogResolver(c ports.Catalog) *CatalogResolver { return &CatalogResolver{Catalog: c} }

// Resolve delegates to Catalog.GetService.
func (r *CatalogResolver) Resolve(ctx context.Context, id core.ServiceID) (*core.Service, error) {
	return r.Catalog.GetService(ctx, id)
}

// NewDeclarativeRuntime creates a runtime that executes declarative HTTP actions.
func NewDeclarativeRuntime(transport ports.Transport, clock ports.Clock, resolver ServiceResolver) *DeclarativeRuntime {
	return &DeclarativeRuntime{transport: transport, clock: clock, resolver: resolver}
}

func (r *DeclarativeRuntime) Execute(ctx context.Context, req ports.ExecuteRequest) (ports.ExecuteResult, error) {
	if req.Action.Request == nil {
		return ports.ExecuteResult{}, fmt.Errorf("declarative runtime: action %s has no request spec", req.Action.ID)
	}
	svc, err := r.resolver.Resolve(ctx, req.Action.Service)
	if err != nil {
		return ports.ExecuteResult{}, fmt.Errorf("resolve service %s: %w", req.Action.Service, err)
	}

	// Pagination loop (no-op for single-page).
	if req.Action.Pagination != nil && req.Action.Pagination.Style == "cursor" {
		return r.executePaginated(ctx, svc, req)
	}
	return r.executeOnce(ctx, svc, req, "")
}

// executeOnce performs a single HTTP request. token is the pagination cursor
// (empty for first/only call).
func (r *DeclarativeRuntime) executeOnce(ctx context.Context, svc *core.Service, req ports.ExecuteRequest, token string) (ports.ExecuteResult, error) {
	httpReq, err := r.buildRequest(ctx, svc, req, token)
	if err != nil {
		return ports.ExecuteResult{}, err
	}

	if req.DryRun {
		return r.dryRunResult(httpReq), nil
	}

	resp, err := r.transport.Do(httpReq)
	if err != nil {
		return ports.ExecuteResult{}, err
	}
	defer resp.Body.Close()

	call := ports.HTTPCall{Method: httpReq.Method, URL: httpReq.URL.String(), Status: resp.StatusCode}

	if resp.StatusCode >= 400 {
		return ports.ExecuteResult{Calls: []ports.HTTPCall{call}}, MapHTTPError(resp, req.Action)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ports.ExecuteResult{Calls: []ports.HTTPCall{call}}, err
	}
	return ports.ExecuteResult{Output: body, Calls: []ports.HTTPCall{call}}, nil
}

// buildRequest constructs the *http.Request from the action spec + inputs +
// auth credential.
func (r *DeclarativeRuntime) buildRequest(ctx context.Context, svc *core.Service, req ports.ExecuteRequest, paginationToken string) (*http.Request, error) {
	sch, err := core.ParseInputSchema(req.Action.InputSchema)
	if err != nil {
		return nil, err
	}
	inputs := sch.ApplyDefaults(req.Inputs)
	groups := sch.ByLocation(inputs)

	// Path interpolation.
	pathVars := map[string]any{}
	for k, v := range groups[core.LocationPath] {
		pathVars[k] = v
	}
	path, err := Interpolate(req.Action.Request.Path, pathVars, true)
	if err != nil {
		return nil, err
	}

	// Base URL + path.
	u, err := url.Parse(svc.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base_url %q: %w", svc.BaseURL, err)
	}
	u.Path = strings.TrimRight(u.Path, "/") + path

	// Query.
	q := u.Query()
	for k, v := range groups[core.LocationQuery] {
		q.Set(k, fmt.Sprint(v))
	}
	if paginationToken != "" && req.Action.Pagination != nil {
		q.Set(req.Action.Pagination.RequestParam, paginationToken)
	}
	u.RawQuery = q.Encode()

	// Body.
	var body io.Reader
	if req.Action.Request.Body != "" {
		bodyBytes, err := r.buildBody(req.Action.Request.Body, groups[core.LocationBody], inputs)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Action.Request.Method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpReq.Header.Set("Accept", "application/json")

	// Custom headers from spec.
	for k, tpl := range req.Action.Request.Headers {
		val, err := Interpolate(tpl, inputs, false)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set(k, val)
	}

	// Header inputs.
	for k, v := range groups[core.LocationHeader] {
		httpReq.Header.Set(k, fmt.Sprint(v))
	}

	// Auth injection.
	if !req.Credential.AccessToken.IsZero() {
		inj, ok := svc.Injection[req.Credential.Provider]
		if !ok {
			// Default: Bearer Authorization header.
			inj = core.AuthInjection{Header: "Authorization", Format: "Bearer {access_token}"}
		}
		if err := InjectAuth(httpReq, req.Credential, inj); err != nil {
			return nil, err
		}
	}
	return httpReq, nil
}

func (r *DeclarativeRuntime) buildBody(template string, bodyInputs map[string]any, allInputs core.Inputs) ([]byte, error) {
	if template == "$inputs" {
		return json.Marshal(bodyInputs)
	}
	interp, err := Interpolate(template, allInputs, false)
	if err != nil {
		return nil, err
	}
	return []byte(interp), nil
}

func (r *DeclarativeRuntime) dryRunResult(req *http.Request) ports.ExecuteResult {
	preview := map[string]any{
		"method":  req.Method,
		"url":     req.URL.String(),
		"headers": filterSensitiveHeaders(req.Header),
	}
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		preview["body"] = string(b)
	}
	out, _ := json.Marshal(preview)
	return ports.ExecuteResult{
		Output: out,
		Calls:  []ports.HTTPCall{{Method: req.Method, URL: req.URL.String(), Status: 0}},
	}
}

func filterSensitiveHeaders(h http.Header) map[string]string {
	out := map[string]string{}
	for k, v := range h {
		lk := strings.ToLower(k)
		if lk == "authorization" || lk == "x-api-key" || lk == "cookie" {
			out[k] = "[REDACTED]"
			continue
		}
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

var _ ports.Runtime = (*DeclarativeRuntime)(nil)
