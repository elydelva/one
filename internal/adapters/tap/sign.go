package tap

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aead.dev/minisign"
)

// Verifier adapts VerifyCatalog to the app.CatalogVerifier interface.
type Verifier struct{}

// NewVerifier returns a Verifier.
func NewVerifier() *Verifier { return &Verifier{} }

// Verify implements app.CatalogVerifier.
func (Verifier) Verify(dir, publicKey string) error { return VerifyCatalog(dir, publicKey) }

// Catalog signature files at the tap repository root.
const (
	checksumFile  = "CATALOG.checksum"
	signatureFile = "CATALOG.minisig"
)

// ErrSignatureMissing is returned when a tap claims to be signed (a pubkey
// was provided) but neither CATALOG.checksum nor CATALOG.minisig is present.
var ErrSignatureMissing = errors.New("tap signature files missing")

// VerifyCatalog verifies the catalog at dir against the supplied minisign
// public key. It enforces two things:
//
//  1. CATALOG.minisig is a valid minisign signature of CATALOG.checksum,
//     produced by the holder of the private key matching pubkeyStr.
//  2. CATALOG.checksum lists exactly the catalog files present under dir
//     (excluding the signature files themselves) with matching sha256 hashes.
//
// pubkeyStr is the textual minisign public key (one-line, base64).
func VerifyCatalog(dir, pubkeyStr string) error {
	pk, err := parsePublicKey(pubkeyStr)
	if err != nil {
		return fmt.Errorf("tap signature: invalid public key: %w", err)
	}
	checksumPath := filepath.Join(dir, checksumFile)
	sigPath := filepath.Join(dir, signatureFile)
	checksum, err := os.ReadFile(checksumPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrSignatureMissing
		}
		return fmt.Errorf("tap signature: read %s: %w", checksumFile, err)
	}
	sig, err := os.ReadFile(sigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrSignatureMissing
		}
		return fmt.Errorf("tap signature: read %s: %w", signatureFile, err)
	}
	if !minisign.Verify(pk, checksum, sig) {
		return errors.New("tap signature: minisign verification failed")
	}
	declared, err := parseChecksumFile(checksum)
	if err != nil {
		return fmt.Errorf("tap signature: parse %s: %w", checksumFile, err)
	}
	observed, err := hashCatalogTree(dir)
	if err != nil {
		return fmt.Errorf("tap signature: hash tree: %w", err)
	}
	return diffChecksums(declared, observed)
}

// parsePublicKey accepts either the one-line minisign public key form
// ("untrusted comment: ..." may precede it) or the bare base64 line.
func parsePublicKey(s string) (minisign.PublicKey, error) {
	s = strings.TrimSpace(s)
	var pk minisign.PublicKey
	if err := pk.UnmarshalText([]byte(s)); err != nil {
		// Fall back: a single base64 line may be missing the "untrusted comment"
		// header that minisign normally writes. Try wrapping it.
		wrapped := "untrusted comment: tap key\n" + s + "\n"
		if err2 := pk.UnmarshalText([]byte(wrapped)); err2 == nil {
			return pk, nil
		}
		return pk, err
	}
	return pk, nil
}

// parseChecksumFile reads lines of "<sha256>  <relpath>" into a map.
// Comment lines (starting with #) and blank lines are ignored.
func parseChecksumFile(raw []byte) (map[string]string, error) {
	out := map[string]string{}
	for i, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// sha256(64 hex) + 2 spaces + path
		if len(line) < 66 || line[64] != ' ' || line[65] != ' ' {
			return nil, fmt.Errorf("line %d: expected '<sha256>  <path>'", i+1)
		}
		sum := strings.ToLower(line[:64])
		if _, err := hex.DecodeString(sum); err != nil {
			return nil, fmt.Errorf("line %d: bad hex digest", i+1)
		}
		path := strings.TrimSpace(line[66:])
		if path == "" {
			return nil, fmt.Errorf("line %d: empty path", i+1)
		}
		if strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
			return nil, fmt.Errorf("line %d: unsafe path %q", i+1, path)
		}
		out[filepath.ToSlash(path)] = sum
	}
	return out, nil
}

// hashCatalogTree walks dir and returns sha256 of every regular file except
// the signature files and any dot-prefixed entries (.git, etc.). Paths in
// the returned map use forward slashes relative to dir.
func hashCatalogTree(root string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		// Skip hidden entries (and their subtrees).
		first := strings.SplitN(rel, string(filepath.Separator), 2)[0]
		if strings.HasPrefix(first, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		// Skip the signature files themselves.
		if rel == checksumFile || rel == signatureFile {
			return nil
		}
		f, err := os.Open(path) //nolint:gosec // G122: walk root is a trusted local tap directory
		if err != nil {
			return err
		}
		h := sha256.New()
		_, copyErr := io.Copy(h, f)
		_ = f.Close()
		if copyErr != nil {
			return copyErr
		}
		out[filepath.ToSlash(rel)] = hex.EncodeToString(h.Sum(nil))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// diffChecksums errors if declared and observed disagree.
func diffChecksums(declared, observed map[string]string) error {
	var missing, extra, mismatched []string
	for path, sum := range declared {
		got, ok := observed[path]
		if !ok {
			missing = append(missing, path)
			continue
		}
		if got != sum {
			mismatched = append(mismatched, path)
		}
	}
	for path := range observed {
		if _, ok := declared[path]; !ok {
			extra = append(extra, path)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	sort.Strings(mismatched)
	if len(missing)+len(extra)+len(mismatched) == 0 {
		return nil
	}
	return fmt.Errorf("tap signature: catalog mismatch (missing=%v, unsigned=%v, mismatched=%v)", missing, extra, mismatched)
}
