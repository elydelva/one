package core

import "fmt"

// ErrNotAuthenticated is returned when no valid credential exists for a service.
type ErrNotAuthenticated struct {
	Service ServiceID
	Account AccountAlias
}

func (e ErrNotAuthenticated) Error() string {
	if e.Account == "" {
		return fmt.Sprintf("not authenticated: %s", e.Service)
	}
	return fmt.Sprintf("not authenticated: %s (account: %s)", e.Service, e.Account)
}

// ErrNotInScope is returned when the requested permission is not granted in .onerc.yaml.
type ErrNotInScope struct {
	Permission Permission
}

func (e ErrNotInScope) Error() string {
	return fmt.Sprintf("not in scope: %s %s", e.Permission.Service, e.Permission.Path)
}

// ErrSetupRequired is returned when a service requires manual setup before use.
type ErrSetupRequired struct {
	Service ServiceID
	Guide   string
	Reason  string
	Human   bool
}

func (e ErrSetupRequired) Error() string {
	return fmt.Sprintf("setup required for %s: %s (guide: %s)", e.Service, e.Reason, e.Guide)
}

// ErrUnknownService is returned when the catalog does not know the requested service.
type ErrUnknownService struct {
	Service ServiceID
}

func (e ErrUnknownService) Error() string {
	return fmt.Sprintf("unknown service: %s", e.Service)
}

// ErrUnknownAction is returned when the service does not have the requested action.
type ErrUnknownAction struct {
	Service ServiceID
	Action  ActionID
}

func (e ErrUnknownAction) Error() string {
	return fmt.Sprintf("unknown action: %s.%s", e.Service, e.Action)
}

// ErrInputValidation is returned when action inputs fail schema validation.
type ErrInputValidation struct {
	Field  string
	Reason string
}

func (e ErrInputValidation) Error() string {
	return fmt.Sprintf("input validation failed: %s — %s", e.Field, e.Reason)
}

// ErrReAuthRequired is returned when a refresh token is expired and the user must re-authenticate.
type ErrReAuthRequired struct {
	Service ServiceID
	Account AccountAlias
}

func (e ErrReAuthRequired) Error() string {
	return fmt.Sprintf("re-authentication required: %s (account: %s)", e.Service, e.Account)
}

// ErrReadOnly is returned when a write is attempted on a read-only vault layer.
type ErrReadOnly struct {
	Source string
}

func (e ErrReadOnly) Error() string {
	if e.Source == "" {
		return "vault is read-only"
	}
	return fmt.Sprintf("vault is read-only: %s", e.Source)
}

// ErrNotSupported is returned when an operation is not supported by an adapter.
type ErrNotSupported struct {
	Source string
	Op     string
}

func (e ErrNotSupported) Error() string {
	return fmt.Sprintf("%s: operation not supported by %s", e.Op, e.Source)
}

// ErrInvalidScopeFile is returned when .onerc.yaml fails to parse or validate.
type ErrInvalidScopeFile struct {
	Path   string
	Reason string
}

func (e ErrInvalidScopeFile) Error() string {
	return fmt.Sprintf("invalid scope file %s: %s", e.Path, e.Reason)
}

// ErrInvalidPattern is returned when a permission pattern is malformed.
type ErrInvalidPattern struct {
	Pattern string
	Reason  string
}

func (e ErrInvalidPattern) Error() string {
	return fmt.Sprintf("invalid permission pattern %q: %s", e.Pattern, e.Reason)
}

// ErrForbidden is returned when the upstream API rejects a request as forbidden (HTTP 403).
// Distinct from ErrNotInScope, which is a local scope refusal.
type ErrForbidden struct {
	Service ServiceID
	Action  ActionID
	Hint    string
}

func (e ErrForbidden) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("forbidden by %s API on action %s: %s", e.Service, e.Action, e.Hint)
	}
	return fmt.Sprintf("forbidden by %s API on action %s", e.Service, e.Action)
}

// ErrNotFound is returned for HTTP 404 from the upstream API.
type ErrNotFound struct {
	Service ServiceID
	Action  ActionID
	Hint    string
	Guide   string // optional install_guide reference
}

func (e ErrNotFound) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("not found (%s %s): %s", e.Service, e.Action, e.Hint)
	}
	return fmt.Sprintf("not found (%s %s)", e.Service, e.Action)
}

// ErrRateLimited is returned for HTTP 429.
type ErrRateLimited struct {
	Service    ServiceID
	RetryAfter string // raw Retry-After header value
}

func (e ErrRateLimited) Error() string {
	if e.RetryAfter != "" {
		return fmt.Sprintf("rate limited by %s (retry after %s)", e.Service, e.RetryAfter)
	}
	return fmt.Sprintf("rate limited by %s", e.Service)
}

// ErrAPIError is returned for HTTP 5xx and any unmapped >= 400 status.
type ErrAPIError struct {
	Service ServiceID
	Status  int
	Body    string
}

func (e ErrAPIError) Error() string {
	return fmt.Sprintf("API error from %s (status %d): %s", e.Service, e.Status, e.Body)
}

// ErrIntegrityCheckFailed is returned when a fetched artifact's hash does not
// match the expected value (e.g. catalog tarball SHA256 mismatch).
type ErrIntegrityCheckFailed struct {
	Source   string
	Expected string
	Actual   string
}

func (e ErrIntegrityCheckFailed) Error() string {
	return fmt.Sprintf("integrity check failed for %s: expected %s, got %s", e.Source, e.Expected, e.Actual)
}

// ErrUnsupportedRuntime is returned when an action requires a runtime not
// available in this version of the binary (e.g. WASM in v0.2).
type ErrUnsupportedRuntime struct {
	Action ActionID
	Reason string
}

func (e ErrUnsupportedRuntime) Error() string {
	return fmt.Sprintf("action %s requires an unsupported runtime: %s", e.Action, e.Reason)
}

// ErrResourceExhausted is returned when a WASM handler exceeds a resource cap
// (memory, CPU time, HTTP calls, log lines, cumulative sleep).
type ErrResourceExhausted struct {
	Action   ActionID
	Resource string
	Cap      string
}

func (e ErrResourceExhausted) Error() string {
	return fmt.Sprintf("handler %s exceeded %s cap (%s)", e.Action, e.Resource, e.Cap)
}

// ErrSandboxViolation is returned when a WASM handler attempts something
// outside its declared allowlist (URL not in calls, credential key not declared,
// fail code not registered, host API version mismatch).
type ErrSandboxViolation struct {
	Action ActionID
	Kind   string
	Detail string
}

func (e ErrSandboxViolation) Error() string {
	return fmt.Sprintf("sandbox violation in %s: %s — %s", e.Action, e.Kind, e.Detail)
}

// ErrHandlerFailed wraps a structured failure emitted by a handler via
// host.fail.withCode.
type ErrHandlerFailed struct {
	Action ActionID
	Code   string
	Msg    string
	Hint   string
}

func (e ErrHandlerFailed) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s: %s (%s) — %s", e.Action, e.Code, e.Msg, e.Hint)
	}
	return fmt.Sprintf("%s: %s (%s)", e.Action, e.Code, e.Msg)
}
