package catalog

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"sort"
	"strings"
	"sync"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"

	pkgcatalog "elydelva/one/pkg/catalog"
)

// IndexFile is the on-CDN representation of the catalog index (<baseURL>/index.json).
type IndexFile struct {
	Version  int                     `json:"version"`
	Services map[string]IndexService `json:"services"`
}

// IndexService describes a single service entry in the index.
type IndexService struct {
	Version       string `json:"version"`
	TarballSha256 string `json:"tarball_sha256"`
	TarballPath   string `json:"tarball_path,omitempty"` // optional; defaults to services/<id>.tar.gz
}

// CatalogHTTP implements ports.Catalog by fetching service definitions from a remote CDN.
// Layout:
//
//	<baseURL>/index.json
//	<baseURL>/services/<id>.tar.gz   (containing service.yaml, actions/*.yaml, SKILL.md, guides/*.md)
//
// Tarballs are integrity-checked against IndexService.TarballSha256 before parsing.
type CatalogHTTP struct {
	baseURL   string
	transport ports.Transport

	mu       sync.Mutex
	bundles  map[core.ServiceID]*serviceBundle
	indexBuf *IndexFile
}

// NewCatalogHTTP creates a CatalogHTTP targeting the given base URL.
func NewCatalogHTTP(baseURL string, transport ports.Transport) *CatalogHTTP {
	return &CatalogHTTP{
		baseURL:   strings.TrimRight(baseURL, "/"),
		transport: transport,
		bundles:   map[core.ServiceID]*serviceBundle{},
	}
}

// serviceBundle is everything we extract from one service tarball.
type serviceBundle struct {
	service *core.Service
	skill   string
	guides  map[string]*core.InstallGuide
}

func (c *CatalogHTTP) GetService(ctx context.Context, id core.ServiceID) (*core.Service, error) {
	b, err := c.bundle(ctx, id)
	if err != nil {
		return nil, err
	}
	svc := *b.service
	return &svc, nil
}

func (c *CatalogHTTP) GetAction(ctx context.Context, svcID core.ServiceID, actionID core.ActionID) (*core.Action, error) {
	svc, err := c.GetService(ctx, svcID)
	if err != nil {
		return nil, err
	}
	for i := range svc.Actions {
		if svc.Actions[i].ID == actionID {
			a := svc.Actions[i]
			return &a, nil
		}
	}
	return nil, core.ErrUnknownAction{Service: svcID, Action: actionID}
}

func (c *CatalogHTTP) ListServices(ctx context.Context) ([]core.Service, error) {
	idx, err := c.index(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(idx.Services))
	for id := range idx.Services {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]core.Service, 0, len(ids))
	for _, id := range ids {
		svc, err := c.GetService(ctx, core.ServiceID(id))
		if err != nil {
			if _, ok := err.(core.ErrUnknownService); ok {
				continue
			}
			return nil, err
		}
		out = append(out, *svc)
	}
	return out, nil
}

func (c *CatalogHTTP) GetSkill(ctx context.Context, id core.ServiceID) (string, error) {
	b, err := c.bundle(ctx, id)
	if err != nil {
		return "", err
	}
	return b.skill, nil
}

func (c *CatalogHTTP) GetGuide(ctx context.Context, svc core.ServiceID, id string) (*core.InstallGuide, error) {
	b, err := c.bundle(ctx, svc)
	if err != nil {
		return nil, err
	}
	g, ok := b.guides[id]
	if !ok {
		return nil, core.ErrNotSupported{Source: "catalog/http", Op: fmt.Sprintf("guide %q not found", id)}
	}
	cp := *g
	return &cp, nil
}

func (c *CatalogHTTP) ListGuides(ctx context.Context, svc core.ServiceID) ([]core.InstallGuide, error) {
	b, err := c.bundle(ctx, svc)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(b.guides))
	for id := range b.guides {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]core.InstallGuide, 0, len(b.guides))
	for _, id := range ids {
		out = append(out, *b.guides[id])
	}
	return out, nil
}

// index fetches and memoizes the catalog index.
func (c *CatalogHTTP) index(ctx context.Context) (*IndexFile, error) {
	c.mu.Lock()
	if c.indexBuf != nil {
		idx := c.indexBuf
		c.mu.Unlock()
		return idx, nil
	}
	c.mu.Unlock()

	raw, err := c.fetch(ctx, c.baseURL+"/index.json")
	if err != nil {
		return nil, err
	}
	var idx IndexFile
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, fmt.Errorf("parse index.json: %w", err)
	}
	if idx.Version != 1 {
		return nil, fmt.Errorf("unsupported index version %d", idx.Version)
	}
	c.mu.Lock()
	c.indexBuf = &idx
	c.mu.Unlock()
	return &idx, nil
}

// bundle fetches, verifies, and parses a service tarball (memoized).
func (c *CatalogHTTP) bundle(ctx context.Context, id core.ServiceID) (*serviceBundle, error) {
	c.mu.Lock()
	if b, ok := c.bundles[id]; ok {
		c.mu.Unlock()
		return b, nil
	}
	c.mu.Unlock()

	idx, err := c.index(ctx)
	if err != nil {
		return nil, err
	}
	entry, ok := idx.Services[string(id)]
	if !ok {
		return nil, core.ErrUnknownService{Service: id}
	}
	tarballPath := entry.TarballPath
	if tarballPath == "" {
		tarballPath = "services/" + string(id) + ".tar.gz"
	}
	tarURL := c.baseURL + "/" + strings.TrimLeft(tarballPath, "/")
	raw, err := c.fetch(ctx, tarURL)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, entry.TarballSha256) {
		return nil, core.ErrIntegrityCheckFailed{Source: tarURL, Expected: entry.TarballSha256, Actual: got}
	}
	b, err := parseBundle(id, raw)
	if err != nil {
		return nil, fmt.Errorf("parse tarball %s: %w", id, err)
	}
	b.service.Version = entry.Version
	b.service.Sha256 = entry.TarballSha256
	c.mu.Lock()
	c.bundles[id] = b
	c.mu.Unlock()
	return b, nil
}

// fetch issues a GET via the transport and returns the body. 404 → ErrUnknownService.
func (c *CatalogHTTP) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.transport.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("catalog/http: GET %s returned %d", url, resp.StatusCode)
	}
	return body, nil
}

// parseBundle extracts service.yaml, actions/*.yaml, SKILL.md, and guides/*.md
// from a gzipped tar archive. Tolerates a single top-level directory wrapper
// (e.g. "github/service.yaml") for either the service id or the reserved dirs.
func parseBundle(id core.ServiceID, raw []byte) (*serviceBundle, error) {
	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	var (
		serviceBytes []byte
		skill        string
		actionFiles  []pkgcatalog.ActionDef
		guides       = map[string]*core.InstallGuide{}
	)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := path.Clean(hdr.Name)
		if strings.Contains(name, "..") {
			return nil, fmt.Errorf("tar entry escapes root: %s", hdr.Name)
		}
		if i := strings.Index(name, "/"); i >= 0 {
			first := name[:i]
			if first == string(id) {
				name = name[i+1:]
			}
		}
		switch {
		case name == "service.yaml":
			serviceBytes, err = io.ReadAll(tr)
			if err != nil {
				return nil, err
			}
		case name == "SKILL.md":
			b, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}
			skill = string(b)
		case strings.HasPrefix(name, "actions/") && strings.HasSuffix(name, ".yaml"):
			b, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}
			fallback := strings.TrimSuffix(path.Base(name), ".yaml")
			a, err := parseActionYAML(b, fallback)
			if err != nil {
				return nil, fmt.Errorf("action %s: %w", name, err)
			}
			actionFiles = append(actionFiles, *a)
		case strings.HasPrefix(name, "guides/") && strings.HasSuffix(name, ".md"):
			b, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}
			guideID := strings.TrimSuffix(path.Base(name), ".md")
			g, err := parseGuideMarkdown(b, guideID, id)
			if err != nil {
				return nil, fmt.Errorf("guide %s: %w", name, err)
			}
			guides[guideID] = g
		}
	}
	if serviceBytes == nil {
		return nil, fmt.Errorf("tarball missing service.yaml")
	}
	def, err := parseServiceYAML(serviceBytes, string(id))
	if err != nil {
		return nil, err
	}
	svc, err := buildService(def, actionFiles)
	if err != nil {
		return nil, err
	}
	return &serviceBundle{service: svc, skill: skill, guides: guides}, nil
}

var _ ports.Catalog = (*CatalogHTTP)(nil)
