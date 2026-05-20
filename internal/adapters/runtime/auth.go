package runtime

import (
	"fmt"
	"net/http"

	"elydelva/one/internal/core"
)

// InjectAuth applies the per-provider auth injection rule to req.
// In v0.2 only Bearer and custom-header injections are supported (PAT, API key).
// OAuth flows are deferred; their access tokens still work via Bearer injection.
func InjectAuth(req *http.Request, cred core.Credential, inj core.AuthInjection) error {
	if inj.Header == "" || inj.Format == "" {
		return fmt.Errorf("missing auth injection rule for provider %s", cred.Provider)
	}
	val, err := Interpolate(inj.Format, map[string]any{
		"access_token":  cred.AccessToken.Reveal(),
		"refresh_token": cred.RefreshToken.Reveal(),
	}, false)
	if err != nil {
		return fmt.Errorf("auth injection: %w", err)
	}
	req.Header.Set(inj.Header, val)
	return nil
}
