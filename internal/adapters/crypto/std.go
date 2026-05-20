package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"

	"elydelva/one/internal/ports"
)

// StdCrypto implements Crypto using the Go standard library.
type StdCrypto struct{}

// NewStdCrypto creates a crypto implementation backed by crypto/rand and crypto/hmac.
func NewStdCrypto() *StdCrypto { return &StdCrypto{} }

func (c *StdCrypto) RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

func (c *StdCrypto) HMACSHA256(key, data []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, key)
	_, err := mac.Write(data)
	if err != nil {
		return nil, err
	}
	return mac.Sum(nil), nil
}

var _ ports.Crypto = (*StdCrypto)(nil)
