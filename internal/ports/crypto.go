package ports

// Crypto provides cryptographic primitives used by the vault and WASM runtime.
type Crypto interface {
	RandomBytes(n int) ([]byte, error)
	HMACSHA256(key, data []byte) ([]byte, error)
}
