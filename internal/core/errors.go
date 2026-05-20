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
