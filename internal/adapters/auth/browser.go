package auth

import (
	"os/exec"
	"runtime"
)

// openBrowser tries to open url in the user's default browser. Best-effort:
// callers should always also print the URL so the user can open it manually.
// Errors are swallowed (returns nil to keep the flow non-blocking).
var openBrowser = func(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url) //nolint:gosec // G204: command name is a constant, only the URL argument varies
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url) //nolint:gosec // G204: command name is a constant, only the URL argument varies
	default:
		cmd = exec.Command("xdg-open", url) //nolint:gosec // G204: command name is a constant, only the URL argument varies
	}
	return cmd.Start()
}
