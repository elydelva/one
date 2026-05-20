package core

import "encoding/json"

// Secret holds sensitive string values. Redacts itself in all serialization paths.
// Call Reveal() only at the HTTP injection point.
type Secret struct {
	v string
}

// NewSecret wraps a plain string in a Secret.
func NewSecret(v string) Secret { return Secret{v: v} }

// Reveal returns the plaintext value. Use only when injecting into HTTP headers or auth flows.
func (s Secret) Reveal() string { return s.v }

// String implements fmt.Stringer — always redacted.
func (s Secret) String() string { return "[REDACTED]" }

// GoString implements fmt.GoStringer — always redacted.
func (s Secret) GoString() string { return `core.Secret("[REDACTED]")` }

// MarshalJSON redacts the value in JSON serialization.
func (s Secret) MarshalJSON() ([]byte, error) { return json.Marshal("[REDACTED]") }

// UnmarshalJSON reads the plaintext from JSON (e.g. from vault storage).
func (s *Secret) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	s.v = v
	return nil
}

// IsZero reports whether the secret is empty.
func (s Secret) IsZero() bool { return s.v == "" }
