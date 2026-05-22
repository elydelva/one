package tap

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"aead.dev/minisign"
)

// makeKeypair generates an ephemeral minisign keypair for tests.
func makeKeypair(t *testing.T) (pubText string, priv minisign.PrivateKey) {
	t.Helper()
	pk, sk, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	text, err := pk.MarshalText()
	if err != nil {
		t.Fatalf("marshal pubkey: %v", err)
	}
	return string(text), sk
}

// writeCatalog creates a tap with service files + a valid signature.
func writeSignedTap(t *testing.T, sk minisign.PrivateKey) (root string) {
	t.Helper()
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "github"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"github/service.yaml": "version: 1\nid: github\nname: GitHub\nbase_url: https://api.github.com\n",
		"github/SKILL.md":     "# github\n",
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	checksum := buildChecksumFile(t, root, files)
	if err := os.WriteFile(filepath.Join(root, checksumFile), checksum, 0o644); err != nil {
		t.Fatal(err)
	}
	sig := minisign.Sign(sk, checksum)
	if err := os.WriteFile(filepath.Join(root, signatureFile), sig, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func buildChecksumFile(t *testing.T, root string, files map[string]string) []byte {
	t.Helper()
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var b strings.Builder
	for _, p := range paths {
		raw, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		b.WriteString(hex.EncodeToString(sum[:]))
		b.WriteString("  ")
		b.WriteString(p)
		b.WriteString("\n")
	}
	return []byte(b.String())
}

func TestVerifyCatalog_HappyPath(t *testing.T) {
	pub, sk := makeKeypair(t)
	root := writeSignedTap(t, sk)
	if err := VerifyCatalog(root, pub); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestVerifyCatalog_WrongKey(t *testing.T) {
	_, sk := makeKeypair(t)
	root := writeSignedTap(t, sk)
	otherPub, _ := makeKeypair(t)
	if err := VerifyCatalog(root, otherPub); err == nil {
		t.Fatal("expected verification failure")
	}
}

func TestVerifyCatalog_MutatedFile(t *testing.T) {
	pub, sk := makeKeypair(t)
	root := writeSignedTap(t, sk)
	// Mutate a signed file after signing.
	if err := os.WriteFile(filepath.Join(root, "github", "service.yaml"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := VerifyCatalog(root, pub)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "mismatched") {
		t.Errorf("expected 'mismatched' in error, got %v", err)
	}
}

func TestVerifyCatalog_ExtraUnsignedFile(t *testing.T) {
	pub, sk := makeKeypair(t)
	root := writeSignedTap(t, sk)
	if err := os.WriteFile(filepath.Join(root, "github", "extra.yaml"), []byte("rogue"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := VerifyCatalog(root, pub)
	if err == nil {
		t.Fatal("expected unsigned-extra error")
	}
	if !strings.Contains(err.Error(), "unsigned") {
		t.Errorf("expected 'unsigned' in error, got %v", err)
	}
}

func TestVerifyCatalog_MissingSignatureFiles(t *testing.T) {
	pub, _ := makeKeypair(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "x", "service.yaml"), []byte("v: 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := VerifyCatalog(root, pub)
	if !errors.Is(err, ErrSignatureMissing) {
		t.Errorf("expected ErrSignatureMissing, got %v", err)
	}
}

func TestVerifyCatalog_IgnoresDotDirs(t *testing.T) {
	pub, sk := makeKeypair(t)
	root := writeSignedTap(t, sk)
	// Simulate a .git directory with random content — must not affect verification.
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyCatalog(root, pub); err != nil {
		t.Errorf("dot-dir leaked into verification: %v", err)
	}
}

func TestParsePublicKey_RoundTrip(t *testing.T) {
	pub, _ := makeKeypair(t)
	if _, err := parsePublicKey(pub); err != nil {
		t.Errorf("parse full form: %v", err)
	}
	// Last non-empty line is the base64 key.
	var bare string
	for _, line := range strings.Split(pub, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "untrusted") {
			bare = line
		}
	}
	if bare == "" {
		t.Skip("no bare key extracted")
	}
	if _, err := parsePublicKey(bare); err != nil {
		t.Errorf("parse bare form: %v", err)
	}
}
