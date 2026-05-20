package cli

import "elydelva/one/internal/core"

// parseProviderKind maps a CLI string to a ProviderKind. Empty string yields PAT.
func parseProviderKind(s string) core.ProviderKind {
	switch s {
	case "", "pat":
		return core.ProviderPAT
	case "api_key", "api-key":
		return core.ProviderAPIKey
	case "oauth2_user":
		return core.ProviderOAuthUser
	case "oauth2_device":
		return core.ProviderOAuthDevice
	case "oauth2_client":
		return core.ProviderOAuthClient
	case "aws_keys":
		return core.ProviderAWSKeys
	default:
		return core.ProviderKind(s)
	}
}
