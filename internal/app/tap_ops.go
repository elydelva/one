// Package app — tap_ops implements the user-tap workflow.
//
// A tap is a GitHub repository (form "user/repo") whose root directory is a
// service catalog laid out identically to the official catalog: each
// top-level directory is one service.
//
// Trust model: TOFU. The first `tap add` prompts for consent and pins the
// upstream commit SHA. Subsequent `tap update` calls warn if the SHA changes
// and require re-consent. The CLI provides the prompt callback; this package
// only enforces the consent gate.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// TapEntry is one registered tap with its pinned upstream commit.
type TapEntry struct {
	Name    string    `json:"name"`            // "user/repo"
	URL     string    `json:"url"`             // https clone URL
	SHA     string    `json:"sha"`             // pinned commit SHA (full)
	AddedAt time.Time `json:"added_at"`
}

// tapRegistry is the on-disk shape of registry.json.
type tapRegistry struct {
	Version int        `json:"version"`
	Taps    []TapEntry `json:"taps"`
}

// TapFetcher abstracts the git operations a tap needs. Production wires the
// os/exec-backed implementation in adapters/tap; tests inject a fake.
type TapFetcher interface {
	// Clone creates dir from a shallow clone of url and returns the HEAD SHA.
	Clone(ctx context.Context, url, dir string) (sha string, err error)
	// HeadSHA returns the current HEAD SHA of an existing clone.
	HeadSHA(ctx context.Context, dir string) (string, error)
	// FetchHead updates dir to the upstream default branch and returns the new SHA.
	FetchHead(ctx context.Context, dir string) (string, error)
}

// ConsentFunc is called by Add and Update to gate trust decisions. It returns
// nil if the user accepts, an error otherwise.
type ConsentFunc func(prompt string) error

// TapOps manages the local tap registry.
type TapOps struct {
	root         string // tap home (default ~/.one/taps)
	fetcher      TapFetcher
	allowedHosts []string // lowercase; empty means default {github.com}
}

// NewTapOps creates a TapOps. root is the directory containing registry.json
// and per-tap clones under <host>/<owner>/<repo>/. allowedHosts restricts
// which git hosts may be cloned (case-insensitive); when nil/empty the
// default is {"github.com"}.
func NewTapOps(root string, fetcher TapFetcher, allowedHosts ...string) *TapOps {
	hosts := make([]string, 0, len(allowedHosts))
	for _, h := range allowedHosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if h != "" {
			hosts = append(hosts, h)
		}
	}
	if len(hosts) == 0 {
		hosts = []string{"github.com"}
	}
	return &TapOps{root: root, fetcher: fetcher, allowedHosts: hosts}
}

// hostAllowed reports whether host is in the allowlist.
func (uc *TapOps) hostAllowed(host string) bool {
	host = strings.ToLower(host)
	for _, h := range uc.allowedHosts {
		if h == host {
			return true
		}
	}
	return false
}

// List returns all registered taps sorted by name.
func (uc *TapOps) List(_ context.Context) ([]TapEntry, error) {
	reg, err := uc.load()
	if err != nil {
		return nil, err
	}
	out := append([]TapEntry(nil), reg.Taps...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// CloneDir returns the on-disk path where a given tap is (or would be) cloned.
//
// For GitHub-style names ("user/repo") the layout is <root>/<user>/<repo>.
// For full https URLs the layout is <root>/<host>/<path-with-slashes-as-dirs>
// (e.g. https://gitlab.com/x/y → <root>/gitlab.com/x/y).
func (uc *TapOps) CloneDir(name string) string {
	res, err := resolveTapTarget(name)
	if err != nil {
		return ""
	}
	return filepath.Join(append([]string{uc.root}, res.dirSegments...)...)
}

// Add clones a new tap, prompts for consent, and persists it. Returns the
// resulting entry or an error if consent is declined, the name is invalid,
// the host is not on the allowlist, or the clone fails.
func (uc *TapOps) Add(ctx context.Context, name string, consent ConsentFunc) (*TapEntry, error) {
	target, err := resolveTapTarget(name)
	if err != nil {
		return nil, err
	}
	if !uc.hostAllowed(target.host) {
		return nil, fmt.Errorf("tap add: host %q not allowed (set ONE_TAP_ALLOWED_HOSTS or use github.com)", target.host)
	}
	reg, err := uc.load()
	if err != nil {
		return nil, err
	}
	for _, t := range reg.Taps {
		if t.Name == target.canonical {
			return nil, fmt.Errorf("tap %s already added (run `one tap update %s` to refresh)", target.canonical, target.canonical)
		}
	}

	cloneURL := target.cloneURL
	dir := uc.CloneDir(target.canonical)
	_ = name // canonical replaces user-supplied form below
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return nil, err
	}
	if _, statErr := os.Stat(dir); statErr == nil {
		if err := os.RemoveAll(dir); err != nil {
			return nil, fmt.Errorf("tap add: clear stale dir: %w", err)
		}
	}
	sha, err := uc.fetcher.Clone(ctx, cloneURL, dir)
	if err != nil {
		return nil, fmt.Errorf("tap add: clone %s: %w", cloneURL, err)
	}

	if consent != nil {
		prompt := fmt.Sprintf("Tap %s pinned to commit %s. Trust this tap?", target.canonical, shortSHA(sha))
		if err := consent(prompt); err != nil {
			_ = os.RemoveAll(dir)
			return nil, err
		}
	}

	entry := TapEntry{Name: target.canonical, URL: cloneURL, SHA: sha, AddedAt: time.Now().UTC()}
	reg.Taps = append(reg.Taps, entry)
	if err := uc.save(reg); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	return &entry, nil
}

// Remove deletes a tap from the registry and removes its clone.
func (uc *TapOps) Remove(_ context.Context, name string) error {
	reg, err := uc.load()
	if err != nil {
		return err
	}
	idx := -1
	for i, t := range reg.Taps {
		if t.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("tap %s not found", name)
	}
	reg.Taps = append(reg.Taps[:idx], reg.Taps[idx+1:]...)
	if err := uc.save(reg); err != nil {
		return err
	}
	if dir := uc.CloneDir(name); dir != "" {
		_ = os.RemoveAll(dir)
	}
	return nil
}

// UpdateResult describes the outcome of an Update call.
type UpdateResult struct {
	Name   string
	OldSHA string
	NewSHA string
	// Changed reports whether the upstream SHA moved. When true, the registry
	// has been updated; consent was required and accepted.
	Changed bool
}

// Update fetches the tap's upstream HEAD and, if the SHA changed, prompts for
// consent before rebinding the pinned SHA.
func (uc *TapOps) Update(ctx context.Context, name string, consent ConsentFunc) (*UpdateResult, error) {
	reg, err := uc.load()
	if err != nil {
		return nil, err
	}
	idx := -1
	for i, t := range reg.Taps {
		if t.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, fmt.Errorf("tap %s not found", name)
	}
	dir := uc.CloneDir(name)
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("tap update: clone dir missing: %w", err)
	}
	newSHA, err := uc.fetcher.FetchHead(ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("tap update: %w", err)
	}
	old := reg.Taps[idx].SHA
	res := &UpdateResult{Name: name, OldSHA: old, NewSHA: newSHA, Changed: newSHA != old}
	if !res.Changed {
		return res, nil
	}
	if consent != nil {
		prompt := fmt.Sprintf("Tap %s: pinned SHA changes from %s to %s. Trust the new revision?",
			name, shortSHA(old), shortSHA(newSHA))
		if err := consent(prompt); err != nil {
			return res, err
		}
	}
	reg.Taps[idx].SHA = newSHA
	if err := uc.save(reg); err != nil {
		return res, err
	}
	return res, nil
}

// load reads the registry, returning a fresh empty registry if the file does
// not exist.
func (uc *TapOps) load() (*tapRegistry, error) {
	path := filepath.Join(uc.root, "registry.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &tapRegistry{Version: 1}, nil
		}
		return nil, err
	}
	var reg tapRegistry
	if err := json.Unmarshal(raw, &reg); err != nil {
		return nil, fmt.Errorf("tap registry: %w", err)
	}
	if reg.Version == 0 {
		reg.Version = 1
	}
	if reg.Version != 1 {
		return nil, fmt.Errorf("tap registry: unsupported version %d", reg.Version)
	}
	return &reg, nil
}

func (uc *TapOps) save(reg *tapRegistry) error {
	if err := os.MkdirAll(uc.root, 0o755); err != nil {
		return err
	}
	reg.Version = 1
	raw, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(uc.root, "registry.json"), raw, 0o644)
}

// tapNameRE matches GitHub-style user/repo. Conservative — refuses dots,
// leading hyphens, and anything that would let a tap escape the tap root.
var tapNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,38}/[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)

// pathSegmentRE matches a single safe path segment for URL-based tap names.
var pathSegmentRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)

// tapTarget is the resolved form of a user-supplied tap reference.
type tapTarget struct {
	canonical   string   // user-visible name stored in the registry
	host        string   // lowercase host
	cloneURL    string   // https URL passed to git clone
	dirSegments []string // path segments under the tap root
}

// resolveTapTarget parses a user-supplied tap reference. Two forms accepted:
//
//   - "user/repo"            → https://github.com/user/repo.git
//   - "https://host/path..." → as-is (must be https + path of safe segments)
//
// Anything else (http, ssh, git://, file://, ..) is rejected.
func resolveTapTarget(name string) (*tapTarget, error) {
	if name == "" {
		return nil, fmt.Errorf("invalid tap name (empty)")
	}
	if strings.HasPrefix(name, "https://") {
		u, err := url.Parse(name)
		if err != nil {
			return nil, fmt.Errorf("invalid tap URL %q: %w", name, err)
		}
		if u.Scheme != "https" || u.Host == "" {
			return nil, fmt.Errorf("invalid tap URL %q (https with host required)", name)
		}
		raw := strings.Trim(u.Path, "/")
		raw = strings.TrimSuffix(raw, ".git")
		if raw == "" {
			return nil, fmt.Errorf("invalid tap URL %q (path required)", name)
		}
		segs := strings.Split(raw, "/")
		if len(segs) < 2 {
			return nil, fmt.Errorf("invalid tap URL %q (need owner/repo path)", name)
		}
		for _, s := range segs {
			if !pathSegmentRE.MatchString(s) {
				return nil, fmt.Errorf("invalid tap URL %q (segment %q rejected)", name, s)
			}
		}
		canonical := strings.ToLower(u.Host) + "/" + strings.Join(segs, "/")
		cloneURL := "https://" + u.Host + "/" + strings.Join(segs, "/") + ".git"
		dirSegs := append([]string{strings.ToLower(u.Host)}, segs...)
		return &tapTarget{canonical: canonical, host: u.Host, cloneURL: cloneURL, dirSegments: dirSegs}, nil
	}
	// Reject schemes we don't accept explicitly to keep error messages clear.
	if i := strings.Index(name, "://"); i >= 0 {
		return nil, fmt.Errorf("invalid tap name %q (only https:// URLs accepted)", name)
	}
	parts := strings.Split(name, "/")
	// Canonical form for non-github taps: "<host>/<owner>/.../<repo>" where
	// the first segment looks like a hostname (contains a dot).
	if len(parts) >= 3 && strings.Contains(parts[0], ".") {
		host := strings.ToLower(parts[0])
		segs := parts[1:]
		for _, s := range segs {
			if !pathSegmentRE.MatchString(s) {
				return nil, fmt.Errorf("invalid tap name %q (segment %q rejected)", name, s)
			}
		}
		canonical := host + "/" + strings.Join(segs, "/")
		cloneURL := "https://" + host + "/" + strings.Join(segs, "/") + ".git"
		dirSegs := append([]string{host}, segs...)
		return &tapTarget{canonical: canonical, host: host, cloneURL: cloneURL, dirSegments: dirSegs}, nil
	}
	if !tapNameRE.MatchString(name) {
		return nil, fmt.Errorf("invalid tap name %q (expected user/repo or https://host/path)", name)
	}
	return &tapTarget{
		canonical:   name,
		host:        "github.com",
		cloneURL:    "https://github.com/" + parts[0] + "/" + parts[1] + ".git",
		dirSegments: []string{parts[0], parts[1]},
	}, nil
}

func shortSHA(sha string) string {
	if len(sha) < 12 {
		return sha
	}
	return sha[:12]
}
