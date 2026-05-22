package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeFetcher records calls and returns scripted SHAs.
type fakeFetcher struct {
	cloneSHA  string
	cloneErr  error
	updateSHA string
	updateErr error
	cloned    []string
	updated   []string
}

func (f *fakeFetcher) Clone(_ context.Context, url, dir string) (string, error) {
	f.cloned = append(f.cloned, url)
	if f.cloneErr != nil {
		return "", f.cloneErr
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	// Create a marker file so tests can verify the dir was populated.
	_ = os.WriteFile(filepath.Join(dir, "service.yaml.marker"), []byte("x"), 0o644)
	return f.cloneSHA, nil
}

func (f *fakeFetcher) HeadSHA(_ context.Context, dir string) (string, error) {
	return f.cloneSHA, nil
}

func (f *fakeFetcher) FetchHead(_ context.Context, dir string) (string, error) {
	f.updated = append(f.updated, dir)
	if f.updateErr != nil {
		return "", f.updateErr
	}
	return f.updateSHA, nil
}

func acceptAll(_ string) error { return nil }
func declineAll(_ string) error {
	return errors.New("declined")
}

func TestTapOps_AddPersistsEntry(t *testing.T) {
	root := t.TempDir()
	uc := NewTapOps(root, &fakeFetcher{cloneSHA: "abc1234567890def1234567890abcdef12345678"})
	entry, err := uc.Add(context.Background(), "acme/extras", acceptAll)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if entry.Name != "acme/extras" || entry.SHA != "abc1234567890def1234567890abcdef12345678" {
		t.Errorf("entry = %+v", entry)
	}
	if entry.URL != "https://github.com/acme/extras.git" {
		t.Errorf("url = %q", entry.URL)
	}
	taps, err := uc.List(context.Background())
	if err != nil || len(taps) != 1 {
		t.Fatalf("list: %v, %v", err, taps)
	}
	if _, err := os.Stat(filepath.Join(root, "acme", "extras", "service.yaml.marker")); err != nil {
		t.Errorf("clone dir missing: %v", err)
	}
}

func TestTapOps_AddDeclinedCleansUp(t *testing.T) {
	root := t.TempDir()
	uc := NewTapOps(root, &fakeFetcher{cloneSHA: "abc"})
	_, err := uc.Add(context.Background(), "acme/extras", declineAll)
	if err == nil {
		t.Fatal("expected error")
	}
	taps, _ := uc.List(context.Background())
	if len(taps) != 0 {
		t.Errorf("declined tap was persisted: %+v", taps)
	}
	if _, err := os.Stat(filepath.Join(root, "acme", "extras")); !os.IsNotExist(err) {
		t.Errorf("clone dir not cleaned: %v", err)
	}
}

func TestTapOps_AddRejectsInvalidName(t *testing.T) {
	uc := NewTapOps(t.TempDir(), &fakeFetcher{})
	cases := []string{"", "no-slash", "../escape/x", "a/b/c", "user/../repo"}
	for _, c := range cases {
		if _, err := uc.Add(context.Background(), c, acceptAll); err == nil {
			t.Errorf("expected error for %q", c)
		}
	}
}

func TestTapOps_AddDuplicate(t *testing.T) {
	uc := NewTapOps(t.TempDir(), &fakeFetcher{cloneSHA: "abc"})
	if _, err := uc.Add(context.Background(), "a/b", acceptAll); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if _, err := uc.Add(context.Background(), "a/b", acceptAll); err == nil {
		t.Errorf("expected duplicate error")
	}
}

func TestTapOps_RemoveDeletesEntryAndClone(t *testing.T) {
	root := t.TempDir()
	uc := NewTapOps(root, &fakeFetcher{cloneSHA: "abc"})
	if _, err := uc.Add(context.Background(), "a/b", acceptAll); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := uc.Remove(context.Background(), "a/b"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	taps, _ := uc.List(context.Background())
	if len(taps) != 0 {
		t.Errorf("taps after remove: %+v", taps)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "b")); !os.IsNotExist(err) {
		t.Errorf("clone not removed: %v", err)
	}
}

func TestTapOps_UpdateUnchanged(t *testing.T) {
	uc := NewTapOps(t.TempDir(), &fakeFetcher{cloneSHA: "abc", updateSHA: "abc"})
	if _, err := uc.Add(context.Background(), "a/b", acceptAll); err != nil {
		t.Fatalf("add: %v", err)
	}
	res, err := uc.Update(context.Background(), "a/b", declineAll) // consent should not be invoked
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if res.Changed {
		t.Errorf("expected no change")
	}
}

func TestTapOps_UpdateChangedRequiresConsent(t *testing.T) {
	uc := NewTapOps(t.TempDir(), &fakeFetcher{cloneSHA: "abc", updateSHA: "def"})
	if _, err := uc.Add(context.Background(), "a/b", acceptAll); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := uc.Update(context.Background(), "a/b", declineAll); err == nil {
		t.Error("expected declined error")
	}
	// Pinned SHA must remain old after decline.
	taps, _ := uc.List(context.Background())
	if taps[0].SHA != "abc" {
		t.Errorf("SHA rebound despite decline: %s", taps[0].SHA)
	}
	// Accepting rebinds.
	res, err := uc.Update(context.Background(), "a/b", acceptAll)
	if err != nil {
		t.Fatalf("accepted update: %v", err)
	}
	if !res.Changed || res.NewSHA != "def" {
		t.Errorf("result: %+v", res)
	}
	taps, _ = uc.List(context.Background())
	if taps[0].SHA != "def" {
		t.Errorf("SHA not rebound: %s", taps[0].SHA)
	}
}

func TestTapOps_AddAcceptsHTTPSURL(t *testing.T) {
	root := t.TempDir()
	uc := NewTapOps(root, &fakeFetcher{cloneSHA: "abc"}, "gitlab.com")
	entry, err := uc.Add(context.Background(), "https://gitlab.com/x/y", acceptAll)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if entry.Name != "gitlab.com/x/y" {
		t.Errorf("canonical name = %q", entry.Name)
	}
	if entry.URL != "https://gitlab.com/x/y.git" {
		t.Errorf("url = %q", entry.URL)
	}
	if _, err := os.Stat(filepath.Join(root, "gitlab.com", "x", "y", "service.yaml.marker")); err != nil {
		t.Errorf("clone dir missing: %v", err)
	}
}

func TestTapOps_AddRejectsNonAllowedHost(t *testing.T) {
	uc := NewTapOps(t.TempDir(), &fakeFetcher{cloneSHA: "abc"}) // default: github.com only
	if _, err := uc.Add(context.Background(), "https://evil.example.com/x/y", acceptAll); err == nil {
		t.Error("expected host-not-allowed error")
	}
}

func TestTapOps_AddRejectsBadSchemes(t *testing.T) {
	uc := NewTapOps(t.TempDir(), &fakeFetcher{cloneSHA: "abc"})
	cases := []string{
		"http://github.com/x/y",
		"ssh://git@github.com/x/y",
		"git://github.com/x/y",
		"file:///etc/passwd",
	}
	for _, c := range cases {
		if _, err := uc.Add(context.Background(), c, acceptAll); err == nil {
			t.Errorf("expected error for %q", c)
		}
	}
}

func TestTapOps_ListEmpty(t *testing.T) {
	uc := NewTapOps(t.TempDir(), &fakeFetcher{})
	taps, err := uc.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(taps) != 0 {
		t.Errorf("expected empty, got %+v", taps)
	}
}
