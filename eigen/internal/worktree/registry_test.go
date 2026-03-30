package worktree

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadRegistry_AbsentFile(t *testing.T) {
	dir := t.TempDir()
	reg, err := ReadRegistry(dir)
	if err != nil {
		t.Fatalf("expected no error for absent file, got: %v", err)
	}
	if len(reg.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(reg.Entries))
	}
}

func TestWriteReadRegistry_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	reg := Registry{
		Entries: []Entry{
			{Name: "feature-foo", Branch: "feature/foo", Path: "/tmp/feature-foo", CreatedAt: now},
		},
	}
	if err := WriteRegistry(dir, reg); err != nil {
		t.Fatalf("WriteRegistry error: %v", err)
	}
	got, err := ReadRegistry(dir)
	if err != nil {
		t.Fatalf("ReadRegistry error: %v", err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got.Entries))
	}
	e := got.Entries[0]
	if e.Name != "feature-foo" {
		t.Errorf("Name = %q, want feature-foo", e.Name)
	}
	if e.Branch != "feature/foo" {
		t.Errorf("Branch = %q, want feature/foo", e.Branch)
	}
	if e.Path != "/tmp/feature-foo" {
		t.Errorf("Path = %q, want /tmp/feature-foo", e.Path)
	}
	if !e.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", e.CreatedAt, now)
	}
}

func TestAddEntry_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	entry := Entry{
		Name:      "test-branch",
		Branch:    "test/branch",
		Path:      "/some/path",
		CreatedAt: time.Now().UTC(),
	}
	if err := AddEntry(dir, entry); err != nil {
		t.Fatalf("AddEntry error: %v", err)
	}
	// Verify .eigen/ was created.
	if _, err := os.Stat(filepath.Join(dir, ".eigen")); err != nil {
		t.Errorf(".eigen dir not created: %v", err)
	}
	// Verify entry is present.
	reg, err := ReadRegistry(dir)
	if err != nil {
		t.Fatalf("ReadRegistry error: %v", err)
	}
	if len(reg.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(reg.Entries))
	}
	if reg.Entries[0].Name != "test-branch" {
		t.Errorf("Name = %q, want test-branch", reg.Entries[0].Name)
	}
}

func TestRemoveEntry_RemovesCorrectEntry(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	reg := Registry{
		Entries: []Entry{
			{Name: "alpha", Branch: "alpha", Path: "/p/alpha", CreatedAt: now},
			{Name: "beta", Branch: "beta", Path: "/p/beta", CreatedAt: now},
			{Name: "gamma", Branch: "gamma", Path: "/p/gamma", CreatedAt: now},
		},
	}
	if err := WriteRegistry(dir, reg); err != nil {
		t.Fatalf("WriteRegistry error: %v", err)
	}

	found, err := RemoveEntry(dir, "beta")
	if err != nil {
		t.Fatalf("RemoveEntry error: %v", err)
	}
	if !found {
		t.Fatal("expected found=true, got false")
	}

	got, err := ReadRegistry(dir)
	if err != nil {
		t.Fatalf("ReadRegistry error: %v", err)
	}
	if len(got.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got.Entries))
	}
	for _, e := range got.Entries {
		if e.Name == "beta" {
			t.Error("beta entry should have been removed")
		}
	}
}

func TestRemoveEntry_NotFound(t *testing.T) {
	dir := t.TempDir()
	reg := Registry{
		Entries: []Entry{
			{Name: "alpha", Branch: "alpha", Path: "/p/alpha", CreatedAt: time.Now().UTC()},
		},
	}
	if err := WriteRegistry(dir, reg); err != nil {
		t.Fatalf("WriteRegistry error: %v", err)
	}

	found, err := RemoveEntry(dir, "nonexistent")
	if err != nil {
		t.Fatalf("RemoveEntry error: %v", err)
	}
	if found {
		t.Error("expected found=false for nonexistent entry")
	}

	// Ensure registry is unchanged.
	got, err := ReadRegistry(dir)
	if err != nil {
		t.Fatalf("ReadRegistry error: %v", err)
	}
	if len(got.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(got.Entries))
	}
}
