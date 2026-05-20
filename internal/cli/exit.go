package cli

import (
	"errors"

	"elydelva/one/internal/core"
)

const (
	ExitOK               = 0
	ExitError            = 1
	ExitNotAuthenticated = 2
	ExitNotInScope       = 3
	ExitSetupRequired    = 4
	ExitUnknownService   = 5
)

// ExitCode maps a typed error to the appropriate exit code.
// Agents should switch on these codes; 0 is success, >0 is a known failure.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}

	var notAuth core.ErrNotAuthenticated
	if errors.As(err, &notAuth) {
		return ExitNotAuthenticated
	}

	var notInScope core.ErrNotInScope
	if errors.As(err, &notInScope) {
		return ExitNotInScope
	}

	var setupRequired core.ErrSetupRequired
	if errors.As(err, &setupRequired) {
		return ExitSetupRequired
	}

	var unknownSvc core.ErrUnknownService
	if errors.As(err, &unknownSvc) {
		return ExitUnknownService
	}

	return ExitError
}
