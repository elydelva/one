package runtime

import (
	"io"
	"net/http"

	"elydelva/one/internal/core"
)

// MapHTTPError converts an HTTP response with status >= 400 into a typed core
// error. Custom hints/install_guide overrides come from action.Errors when present.
//
// The response body is consumed (read fully and closed by the caller).
func MapHTTPError(resp *http.Response, action core.Action) error {
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	spec, hasSpec := action.Errors[resp.StatusCode]

	switch resp.StatusCode {
	case http.StatusUnauthorized: // 401
		return core.ErrNotAuthenticated{Service: action.Service}
	case http.StatusForbidden: // 403
		return core.ErrForbidden{Service: action.Service, Action: action.ID, Hint: hint(spec, hasSpec, "")}
	case http.StatusNotFound: // 404
		return core.ErrNotFound{
			Service: action.Service, Action: action.ID,
			Hint:    hint(spec, hasSpec, ""),
			Guide:   guide(spec, hasSpec),
		}
	case http.StatusTooManyRequests: // 429
		return core.ErrRateLimited{Service: action.Service, RetryAfter: resp.Header.Get("Retry-After")}
	}

	if resp.StatusCode >= 500 {
		return core.ErrAPIError{Service: action.Service, Status: resp.StatusCode, Body: bodyStr}
	}
	// Any unmapped 4xx → generic API error.
	return core.ErrAPIError{Service: action.Service, Status: resp.StatusCode, Body: bodyStr}
}

func hint(spec core.ErrorSpec, ok bool, fallback string) string {
	if ok && spec.Hint != "" {
		return spec.Hint
	}
	return fallback
}

func guide(spec core.ErrorSpec, ok bool) string {
	if ok {
		return spec.InstallGuide
	}
	return ""
}
