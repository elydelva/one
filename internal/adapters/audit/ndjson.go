// Package audit provides Audit port adapters.
package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// NDJSONAudit is an append-only NDJSON audit log with monthly file rotation.
// Files are stored at <dir>/audit-YYYY-MM.ndjson. Reads merge across files.
type NDJSONAudit struct {
	dir       string
	retention time.Duration // files older than retention are pruned on Record
	mu        sync.Mutex
}

// NewNDJSONAudit creates an NDJSON audit log. dir is the directory to write into.
// retention is the max age for retained log files (default 30 days when zero).
func NewNDJSONAudit(dir string, retention time.Duration) *NDJSONAudit {
	if retention <= 0 {
		retention = 30 * 24 * time.Hour
	}
	return &NDJSONAudit{dir: dir, retention: retention}
}

// Record appends an event to the current month's file. fsync per event.
func (a *NDJSONAudit) Record(_ context.Context, ev core.AuditEvent) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ev.At.IsZero() {
		ev.At = time.Now().UTC()
	}
	if err := os.MkdirAll(a.dir, 0o700); err != nil {
		return fmt.Errorf("audit: mkdir: %w", err)
	}
	path := a.fileFor(ev.At)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("audit: open: %w", err)
	}
	defer func() { _ = f.Close() }()
	buf, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("audit: marshal: %w", err)
	}
	buf = append(buf, '\n')
	if _, err := f.Write(buf); err != nil {
		return fmt.Errorf("audit: write: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("audit: sync: %w", err)
	}
	a.pruneOld(ev.At)
	return nil
}

// Read returns events matching f, ordered most-recent first. Limit 0 = no cap.
func (a *NDJSONAudit) Read(_ context.Context, f core.AuditFilter) ([]core.AuditEvent, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	files, err := a.listFiles()
	if err != nil {
		return nil, err
	}
	// Read newest file first.
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	var out []core.AuditEvent
	for _, path := range files {
		evs, err := a.readFile(path)
		if err != nil {
			return out, err
		}
		// Iterate newest-first within file.
		for i := len(evs) - 1; i >= 0; i-- {
			ev := evs[i]
			if !matches(ev, f) {
				continue
			}
			out = append(out, ev)
			if f.Limit > 0 && len(out) >= f.Limit {
				return out, nil
			}
		}
	}
	return out, nil
}

func (a *NDJSONAudit) fileFor(t time.Time) string {
	return filepath.Join(a.dir, fmt.Sprintf("audit-%04d-%02d.ndjson", t.Year(), int(t.Month())))
}

func (a *NDJSONAudit) listFiles() ([]string, error) {
	entries, err := os.ReadDir(a.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("audit: readdir: %w", err)
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "audit-") && strings.HasSuffix(name, ".ndjson") {
			out = append(out, filepath.Join(a.dir, name))
		}
	}
	return out, nil
}

func (a *NDJSONAudit) readFile(path string) ([]core.AuditEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("audit: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []core.AuditEvent
	for scanner.Scan() {
		var ev core.AuditEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue // skip corrupted line
		}
		out = append(out, ev)
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return out, fmt.Errorf("audit: scan %s: %w", path, err)
	}
	return out, nil
}

func (a *NDJSONAudit) pruneOld(now time.Time) {
	files, err := a.listFiles()
	if err != nil {
		return
	}
	cutoff := now.Add(-a.retention)
	for _, p := range files {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(p)
		}
	}
}

func matches(ev core.AuditEvent, f core.AuditFilter) bool {
	if !f.Since.IsZero() && ev.At.Before(f.Since) {
		return false
	}
	if f.Service != "" && ev.Service != f.Service {
		return false
	}
	if f.Kind != "" && ev.Kind != f.Kind {
		return false
	}
	if f.TraceID != "" && ev.TraceID != f.TraceID {
		return false
	}
	return true
}

var _ ports.Audit = (*NDJSONAudit)(nil)
