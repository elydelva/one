package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DefaultUpgradeBaseURL is the canonical release host. Override via env in tests.
const DefaultUpgradeBaseURL = "https://github.com/elydelva/one/releases/download"

// UpgradeInput holds parameters for the Upgrade use case.
type UpgradeInput struct {
	TargetVersion string // e.g. "v1.0.0"; required
	DryRun        bool
	BaseURL       string // override (test/dev)
	BinaryPath    string // override (test) — defaults to os.Executable()
	HTTPGet       func(url string) (*http.Response, error)
}

// UpgradeOutput describes the planned or completed upgrade.
type UpgradeOutput struct {
	From    string
	To      string
	URL     string
	SHA256  string
	Applied bool
}

// Upgrade replaces the current binary with the target release version.
// Verifies a SHA256 sidecar (<asset>.sha256) before swapping atomically.
type Upgrade struct {
	currentVersion string
}

// NewUpgrade creates an Upgrade use case bound to the running binary's version.
func NewUpgrade(currentVersion string) *Upgrade {
	return &Upgrade{currentVersion: currentVersion}
}

// Run performs the upgrade flow.
func (uc *Upgrade) Run(_ context.Context, in UpgradeInput) (UpgradeOutput, error) {
	out := UpgradeOutput{From: uc.currentVersion, To: in.TargetVersion}
	if in.TargetVersion == "" {
		return out, errors.New("upgrade: target version required")
	}
	binaryPath := in.BinaryPath
	if binaryPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return out, fmt.Errorf("upgrade: locate binary: %w", err)
		}
		binaryPath = exe
	}
	if strings.Contains(binaryPath, "/go/pkg/") || strings.Contains(binaryPath, "/go-build/") {
		return out, errors.New("upgrade: binary appears installed via `go install` — use your package manager")
	}
	base := in.BaseURL
	if base == "" {
		base = DefaultUpgradeBaseURL
	}
	asset := fmt.Sprintf("one_%s_%s_%s.tar.gz", strings.TrimPrefix(in.TargetVersion, "v"), runtime.GOOS, runtime.GOARCH)
	assetURL := fmt.Sprintf("%s/%s/%s", base, in.TargetVersion, asset)
	sumURL := assetURL + ".sha256"
	out.URL = assetURL

	get := in.HTTPGet
	if get == nil {
		get = http.Get
	}
	expectedSum, err := fetchSum(get, sumURL)
	if err != nil {
		return out, err
	}
	out.SHA256 = expectedSum
	if in.DryRun {
		return out, nil
	}
	tmp, err := downloadAndVerify(get, assetURL, expectedSum, filepath.Dir(binaryPath))
	if err != nil {
		return out, err
	}
	defer os.Remove(tmp)
	if err := os.Rename(tmp, binaryPath); err != nil {
		return out, fmt.Errorf("upgrade: replace binary: %w", err)
	}
	out.Applied = true
	return out, nil
}

func fetchSum(get func(string) (*http.Response, error), url string) (string, error) {
	resp, err := get(url)
	if err != nil {
		return "", fmt.Errorf("upgrade: fetch sum: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upgrade: sum http %d", resp.StatusCode)
	}
	buf, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", err
	}
	// Sidecar format: "<hex>  <filename>\n" — take first whitespace-separated token.
	fields := strings.Fields(string(buf))
	if len(fields) == 0 {
		return "", errors.New("upgrade: empty sum file")
	}
	if len(fields[0]) != 64 {
		return "", fmt.Errorf("upgrade: unexpected sum length %d", len(fields[0]))
	}
	return strings.ToLower(fields[0]), nil
}

func downloadAndVerify(get func(string) (*http.Response, error), url, expected, dir string) (string, error) {
	resp, err := get(url)
	if err != nil {
		return "", fmt.Errorf("upgrade: download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upgrade: download http %d", resp.StatusCode)
	}
	tmp, err := os.CreateTemp(dir, ".one-upgrade-*")
	if err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expected {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("upgrade: sha256 mismatch (want %s, got %s)", expected, got)
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}
