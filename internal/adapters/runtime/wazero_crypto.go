//go:build !nowasm

package runtime

import (
	"context"
	"crypto/hmac"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// registerCrypto wires host.crypto.* primitives onto a host module builder.
func registerCrypto(b wazero.HostModuleBuilder) {
	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, inPtr, inLen, outPtr uint32) {
		buf, ok := m.Memory().Read(inPtr, inLen)
		if !ok {
			return
		}
		sum := sha256.Sum256(buf)
		m.Memory().Write(outPtr, sum[:])
	}).Export("crypto_sha256")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, inPtr, inLen, outPtr uint32) {
		buf, ok := m.Memory().Read(inPtr, inLen)
		if !ok {
			return
		}
		sum := sha512.Sum512(buf)
		m.Memory().Write(outPtr, sum[:])
	}).Export("crypto_sha512")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, keyPtr, keyLen, dataPtr, dataLen, outPtr uint32) {
		key, ok := m.Memory().Read(keyPtr, keyLen)
		if !ok {
			return
		}
		data, ok := m.Memory().Read(dataPtr, dataLen)
		if !ok {
			return
		}
		mac := hmac.New(sha256.New, key)
		mac.Write(data)
		m.Memory().Write(outPtr, mac.Sum(nil))
	}).Export("crypto_hmac_sha256")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, keyPtr, keyLen, dataPtr, dataLen, outPtr uint32) {
		key, ok := m.Memory().Read(keyPtr, keyLen)
		if !ok {
			return
		}
		data, ok := m.Memory().Read(dataPtr, dataLen)
		if !ok {
			return
		}
		mac := hmac.New(sha512.New, key)
		mac.Write(data)
		m.Memory().Write(outPtr, mac.Sum(nil))
	}).Export("crypto_hmac_sha512")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, n, outPtr uint32) uint32 {
		buf := make([]byte, n)
		if _, err := cryptorand.Read(buf); err != nil {
			return 0
		}
		m.Memory().Write(outPtr, buf)
		return n
	}).Export("crypto_random_bytes")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, outPtr uint32) uint32 {
		u, err := uuid.NewRandom()
		if err != nil {
			return 0
		}
		s := u.String()
		m.Memory().Write(outPtr, []byte(s))
		return uint32(len(s))
	}).Export("crypto_uuid_v4")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, inPtr, inLen, outPtr, outMax, urlSafe uint32) uint32 {
		buf, ok := m.Memory().Read(inPtr, inLen)
		if !ok {
			return 0
		}
		var s string
		if urlSafe != 0 {
			s = base64.RawURLEncoding.EncodeToString(buf)
		} else {
			s = base64.StdEncoding.EncodeToString(buf)
		}
		if uint32(len(s)) > outMax {
			return 0
		}
		m.Memory().Write(outPtr, []byte(s))
		return uint32(len(s))
	}).Export("crypto_base64_encode")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, inPtr, inLen, outPtr, outMax, urlSafe uint32) uint32 {
		buf, ok := m.Memory().Read(inPtr, inLen)
		if !ok {
			return 0
		}
		var (
			out []byte
			err error
		)
		if urlSafe != 0 {
			out, err = base64.RawURLEncoding.DecodeString(string(buf))
		} else {
			out, err = base64.StdEncoding.DecodeString(string(buf))
		}
		if err != nil || uint32(len(out)) > outMax {
			return 0
		}
		m.Memory().Write(outPtr, out)
		return uint32(len(out))
	}).Export("crypto_base64_decode")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, inPtr, inLen, outPtr, outMax uint32) uint32 {
		buf, ok := m.Memory().Read(inPtr, inLen)
		if !ok {
			return 0
		}
		s := hex.EncodeToString(buf)
		if uint32(len(s)) > outMax {
			return 0
		}
		m.Memory().Write(outPtr, []byte(s))
		return uint32(len(s))
	}).Export("crypto_hex_encode")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, inPtr, inLen, outPtr, outMax uint32) uint32 {
		buf, ok := m.Memory().Read(inPtr, inLen)
		if !ok {
			return 0
		}
		out, err := hex.DecodeString(string(buf))
		if err != nil || uint32(len(out)) > outMax {
			return 0
		}
		m.Memory().Write(outPtr, out)
		return uint32(len(out))
	}).Export("crypto_hex_decode")
}

// _ keeps fmt referenced for future debug strings without breaking the build.
var _ = fmt.Sprintf
