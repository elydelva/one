package auth

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// CertificateProvider stores a PEM client certificate + private key for mTLS.
// Login reads PEMs from filesystem paths supplied by env vars (no prompt: PEMs
// are too large to paste comfortably).
//
//	ONE_CERT_<SERVICE>_<ACCOUNT>_CERT=/path/to/cert.pem
//	ONE_CERT_<SERVICE>_<ACCOUNT>_KEY=/path/to/key.pem
//
// The credential is stored as:
//
//	AccessToken  = cert PEM bytes
//	RefreshToken = key PEM bytes
type CertificateProvider struct{}

// NewCertificateProvider creates an mTLS credential provider.
func NewCertificateProvider() *CertificateProvider { return &CertificateProvider{} }

func (p *CertificateProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderCertificate
}

func (p *CertificateProvider) Login(_ context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error) {
	envPrefix := fmt.Sprintf("ONE_CERT_%s_%s_", normalizeEnv(string(svc)), normalizeEnv(string(alias)))
	certPath := os.Getenv(envPrefix + "CERT")
	keyPath := os.Getenv(envPrefix + "KEY")
	if certPath == "" || keyPath == "" {
		return core.Credential{}, fmt.Errorf("certificate: set %sCERT and %sKEY env vars", envPrefix, envPrefix)
	}
	certPEM, err := os.ReadFile(certPath) //nolint:gosec // G703: path comes from trusted local user config
	if err != nil {
		return core.Credential{}, fmt.Errorf("certificate: read %s: %w", certPath, err)
	}
	keyPEM, err := os.ReadFile(keyPath) //nolint:gosec // G703: path comes from trusted local user config
	if err != nil {
		return core.Credential{}, fmt.Errorf("certificate: read %s: %w", keyPath, err)
	}
	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return core.Credential{}, fmt.Errorf("certificate: invalid PEM pair: %w", err)
	}
	return core.Credential{
		Provider:     core.ProviderCertificate,
		Service:      svc,
		Account:      alias,
		AccessToken:  core.NewSecret(string(certPEM)),
		RefreshToken: core.NewSecret(string(keyPEM)),
	}, nil
}

func (p *CertificateProvider) Refresh(_ context.Context, cred core.Credential) (core.Credential, error) {
	return cred, nil // certificates rotate via re-login
}

// CertPair returns the PEM bytes embedded in the credential.
// Used by transport-aware runtimes that need to build a tls.Config.
func CertPair(cred core.Credential) (cert, key []byte, err error) {
	if cred.Provider != core.ProviderCertificate {
		return nil, nil, errors.New("certificate: not a certificate credential")
	}
	return []byte(cred.AccessToken.Reveal()), []byte(cred.RefreshToken.Reveal()), nil
}

// MarshalAccountIndex serializes the env-driven account refs for debugging.
// Helper exported for `one vault status`.
func MarshalAccountIndex(refs []core.AccountRef) ([]byte, error) {
	return json.Marshal(refs)
}

func normalizeEnv(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z':
			out = append(out, c-32)
		case c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

var _ ports.AuthProvider = (*CertificateProvider)(nil)
