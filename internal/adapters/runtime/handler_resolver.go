//go:build !nowasm

package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"elydelva/one/internal/core"
)

// FSHandlerResolver loads handler bytes from a local catalog directory.
// It refuses paths escaping the per-service handler directory.
type FSHandlerResolver struct{ root string }

// NewFSHandlerResolver returns a resolver rooted at the catalog directory.
func NewFSHandlerResolver(catalogRoot string) *FSHandlerResolver {
	return &FSHandlerResolver{root: catalogRoot}
}

// ReadHandler implements HandlerResolver.
func (r *FSHandlerResolver) ReadHandler(_ context.Context, svc core.ServiceID, file string) ([]byte, error) {
	if file == "" {
		return nil, fmt.Errorf("handler file path is empty")
	}
	base := filepath.Join(r.root, string(svc))
	full := filepath.Clean(filepath.Join(base, file))
	rel, err := filepath.Rel(base, full)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(file) {
		return nil, fmt.Errorf("handler path escapes service dir: %s", file)
	}
	return os.ReadFile(full)
}
