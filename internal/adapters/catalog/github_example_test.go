package catalog

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestGitHubFullAccessExampleAllowsEveryCatalogPermission(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	path := filepath.Join(root, "examples", "github-full-access", ".onerc.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read full-access example: %v", err)
	}
	var example struct {
		Services map[string]struct {
			Allow []string `yaml:"allow"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(raw, &example); err != nil {
		t.Fatalf("parse full-access example: %v", err)
	}
	githubScope, ok := example.Services["github"]
	if !ok {
		t.Fatal("full-access example has no github service scope")
	}
	allowed := make(map[string]bool, len(githubScope.Allow))
	for _, permission := range githubScope.Allow {
		allowed[permission] = true
	}

	service, err := NewCatalogEmbed().GetService(context.Background(), "github")
	if err != nil {
		t.Fatalf("load embedded GitHub catalog: %v", err)
	}
	for _, action := range service.Actions {
		if permission := string(action.Permission); !allowed[permission] {
			t.Errorf("full-access example does not allow permission %q used by %s", permission, action.ID)
		}
	}
}
