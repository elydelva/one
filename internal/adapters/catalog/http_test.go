package catalog

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"elydelva/one/internal/adapters/transport"
	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/portstest"
)

// buildTarball builds a gzipped tar from a map of path → content.
func buildTarball(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const githubServiceYAML = `version: 1
id: github
name: GitHub
base_url: https://api.github.com
auth:
  providers:
    - pat
  injection:
    pat:
      header: Authorization
      format: "Bearer {access_token}"
`

const issuesReadActionYAML = `id: issues.read
description: Read one issue
permission: issues.read
request:
  method: GET
  path: /repos/{owner}/{repo}/issues/{number}
`

const sampleGuideMD = `---
title: Get a personal access token
verify:
  action: issues.read
  hint: try one github issues.read
---
# Setup

1. Visit https://github.com/settings/tokens
`

// newGithubServer starts an httptest server serving a canonical github bundle.
func newGithubServer(t *testing.T, tamper bool) *httptest.Server {
	t.Helper()
	tarball := buildTarball(t, map[string]string{
		"service.yaml":             githubServiceYAML,
		"actions/issues.read.yaml": issuesReadActionYAML,
		"SKILL.md":                 "# github skill",
		"guides/setup.md":          sampleGuideMD,
	})
	sum := sha256.Sum256(tarball)
	hash := hex.EncodeToString(sum[:])
	if tamper {
		hash = strings.Repeat("0", 64)
	}
	idx := IndexFile{
		Version: 1,
		Services: map[string]IndexService{
			"github": {Version: "0.4.2", TarballSha256: hash},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(idx)
	})
	mux.HandleFunc("/services/github.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(tarball)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// newTransport returns a NetHTTP transport that trusts the httptest host (loopback).
func newTransport(t *testing.T, host string) ports.Transport {
	t.Helper()
	return transport.NewNetHTTP(transport.WithAllowHTTP(true), transport.WithAllowedHosts(host))
}

func TestCatalogHTTP_Contract(t *testing.T) {
	portstest.RunCatalogTests(t, "CatalogHTTP", func(t *testing.T) ports.Catalog {
		srv := newGithubServer(t, false)
		host := hostOnly(srv.URL)
		return NewCatalogHTTP(srv.URL, newTransport(t, host))
	})
}

func TestCatalogHTTP_TamperedTarballRejected(t *testing.T) {
	srv := newGithubServer(t, true)
	host := hostOnly(srv.URL)
	c := NewCatalogHTTP(srv.URL, newTransport(t, host))
	_, err := c.GetService(context.Background(), "github")
	var integ core.ErrIntegrityCheckFailed
	if err == nil {
		t.Fatal("expected integrity error")
	}
	if !asAny(err, &integ) {
		t.Fatalf("got %T: %v", err, err)
	}
}

func TestCatalogHTTP_GuideAndSkillFromTarball(t *testing.T) {
	srv := newGithubServer(t, false)
	host := hostOnly(srv.URL)
	c := NewCatalogHTTP(srv.URL, newTransport(t, host))

	skill, err := c.GetSkill(context.Background(), "github")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(skill, "github skill") {
		t.Errorf("skill = %q", skill)
	}

	g, err := c.GetGuide(context.Background(), "github", "setup")
	if err != nil {
		t.Fatal(err)
	}
	if g.Title != "Get a personal access token" {
		t.Errorf("title = %q", g.Title)
	}
	if g.Verify == nil || g.Verify.Action != "issues.read" {
		t.Errorf("verify = %+v", g.Verify)
	}
	if !strings.Contains(g.Content, "# Setup") {
		t.Errorf("content missing body: %q", g.Content)
	}

	svc, err := c.GetService(context.Background(), "github")
	if err != nil {
		t.Fatal(err)
	}
	if svc.Version != "0.4.2" {
		t.Errorf("version = %q", svc.Version)
	}
}

func asAny(err error, target any) bool { return errors.As(err, target) }

// hostOnly extracts the host (no port) from a URL like "http://127.0.0.1:34567".
func hostOnly(rawURL string) string {
	s := strings.TrimPrefix(rawURL, "http://")
	if i := strings.Index(s, ":"); i >= 0 {
		return s[:i]
	}
	return s
}
