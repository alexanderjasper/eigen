package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/alexanderjasper/eigen/internal/spec"
)

// writeChangeFile writes a Change as YAML into dir with the given slug.
func writeChangeFile(t *testing.T, dir string, ch spec.Change, slug string) {
	t.Helper()
	data, err := yaml.Marshal(ch)
	if err != nil {
		t.Fatalf("marshal change: %v", err)
	}
	filename := filepath.Join(dir, fmt.Sprintf("%03d_%s.yaml", ch.Sequence, slug))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		t.Fatalf("write change file: %v", err)
	}
}

// writeSpecFile writes a SpecModule as YAML into specsRoot/modPath/spec.yaml.
func writeSpecFile(t *testing.T, specsRoot, modPath string, s spec.SpecModule) {
	t.Helper()
	dir := filepath.Join(specsRoot, filepath.FromSlash(modPath))
	data, err := yaml.Marshal(s)
	if err != nil {
		t.Fatalf("marshal spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.yaml"), data, 0644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}
}

// setupModule creates the module directory and its changes/ subdirectory.
func setupModule(t *testing.T, specsRoot, modPath string) string {
	t.Helper()
	dir := filepath.Join(specsRoot, filepath.FromSlash(modPath))
	changesDir := filepath.Join(dir, "changes")
	if err := os.MkdirAll(changesDir, 0755); err != nil {
		t.Fatalf("setup module: %v", err)
	}
	return changesDir
}

func TestModulePath(t *testing.T) {
	t.Run("correct_absolute_path", func(t *testing.T) {
		got := ModulePath("/tmp/specs", "spec-cli/cmd-new")
		want := filepath.Join("/tmp/specs", "spec-cli", "cmd-new")
		if got != want {
			t.Errorf("ModulePath = %q, want %q", got, want)
		}
	})
}

func TestChangesPath(t *testing.T) {
	t.Run("changes_subdirectory", func(t *testing.T) {
		got := ChangesPath("/tmp/specs", "spec-cli/cmd-new")
		want := filepath.Join("/tmp/specs", "spec-cli", "cmd-new", "changes")
		if got != want {
			t.Errorf("ChangesPath = %q, want %q", got, want)
		}
	})
}

func TestSpecPath(t *testing.T) {
	t.Run("spec_yaml_path", func(t *testing.T) {
		got := SpecPath("/tmp/specs", "spec-cli/cmd-new")
		want := filepath.Join("/tmp/specs", "spec-cli", "cmd-new", "spec.yaml")
		if got != want {
			t.Errorf("SpecPath = %q, want %q", got, want)
		}
	})
}

func TestReadChanges(t *testing.T) {
	t.Run("parses_all_yaml_files", func(t *testing.T) {
		root := t.TempDir()
		changesDir := setupModule(t, root, "mymod")

		writeChangeFile(t, changesDir, spec.Change{
			ID: "chg-001", Sequence: 1, Summary: "first",
		}, "first")
		writeChangeFile(t, changesDir, spec.Change{
			ID: "chg-002", Sequence: 2, Summary: "second",
		}, "second")

		got, err := ReadChanges(root, "mymod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
	})

	t.Run("skips_non_yaml_and_subdirs", func(t *testing.T) {
		root := t.TempDir()
		changesDir := setupModule(t, root, "mymod")

		writeChangeFile(t, changesDir, spec.Change{
			ID: "chg-001", Sequence: 1, Summary: "valid",
		}, "valid")

		// Write a .txt file
		if err := os.WriteFile(filepath.Join(changesDir, "readme.txt"), []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}
		// Create a subdirectory
		if err := os.Mkdir(filepath.Join(changesDir, "subdir"), 0755); err != nil {
			t.Fatal(err)
		}

		got, err := ReadChanges(root, "mymod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].ID != "chg-001" {
			t.Errorf("ID = %q, want chg-001", got[0].ID)
		}
	})

	t.Run("sorted_by_sequence", func(t *testing.T) {
		root := t.TempDir()
		changesDir := setupModule(t, root, "mymod")

		writeChangeFile(t, changesDir, spec.Change{ID: "c3", Sequence: 3}, "third")
		writeChangeFile(t, changesDir, spec.Change{ID: "c1", Sequence: 1}, "first")
		writeChangeFile(t, changesDir, spec.Change{ID: "c2", Sequence: 2}, "second")

		got, err := ReadChanges(root, "mymod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("len = %d, want 3", len(got))
		}
		for i, wantSeq := range []int{1, 2, 3} {
			if got[i].Sequence != wantSeq {
				t.Errorf("got[%d].Sequence = %d, want %d", i, got[i].Sequence, wantSeq)
			}
		}
	})

	t.Run("error_missing_dir", func(t *testing.T) {
		root := t.TempDir()
		_, err := ReadChanges(root, "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("error_malformed_yaml", func(t *testing.T) {
		root := t.TempDir()
		changesDir := setupModule(t, root, "mymod")

		if err := os.WriteFile(filepath.Join(changesDir, "001_bad.yaml"), []byte(":\n  :\n    - :\n  {{{"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ReadChanges(root, "mymod")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestWriteSpec(t *testing.T) {
	t.Run("valid_roundtrip_yaml", func(t *testing.T) {
		root := t.TempDir()
		setupModule(t, root, "mymod")

		original := spec.SpecModule{
			ID:          "d/m",
			Domain:      "d",
			Module:      "m",
			Owner:       "alice",
			Title:       "My Spec",
			Status:      "draft",
			Description: "desc",
			Behavior:    "beh",
			Dependencies: []string{"dep-a"},
			Technology:  map[string]string{"lang": "go"},
		}

		if err := WriteSpec(root, "mymod", original); err != nil {
			t.Fatalf("WriteSpec error: %v", err)
		}

		// Verify file exists and has correct mode
		info, err := os.Stat(SpecPath(root, "mymod"))
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0644 {
			t.Errorf("perm = %o, want 0644", perm)
		}

		// Round-trip
		got, err := ReadSpec(root, "mymod")
		if err != nil {
			t.Fatalf("ReadSpec error: %v", err)
		}
		if got.ID != original.ID {
			t.Errorf("ID = %q, want %q", got.ID, original.ID)
		}
		if got.Title != original.Title {
			t.Errorf("Title = %q, want %q", got.Title, original.Title)
		}
		if got.Owner != original.Owner {
			t.Errorf("Owner = %q, want %q", got.Owner, original.Owner)
		}
	})
}

func TestWriteChange(t *testing.T) {
	t.Run("correct_filename_format", func(t *testing.T) {
		root := t.TempDir()
		setupModule(t, root, "mymod")

		ch := spec.Change{ID: "chg-007", Sequence: 7, Summary: "feature"}
		if err := WriteChange(root, "mymod", ch, "add-feature"); err != nil {
			t.Fatalf("WriteChange error: %v", err)
		}

		path := filepath.Join(ChangesPath(root, "mymod"), "007_add-feature.yaml")
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("file not found: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0644 {
			t.Errorf("perm = %o, want 0644", perm)
		}
	})

	t.Run("roundtrip_yaml", func(t *testing.T) {
		root := t.TempDir()
		setupModule(t, root, "mymod")

		original := spec.Change{
			ID:        "chg-001",
			Sequence:  1,
			Timestamp: "2026-01-01T00:00:00Z",
			Author:    "bob",
			Type:      "created",
			Summary:   "initial",
			Reason:    "because",
			Status:    "draft",
			Changes: spec.ChangeSet{
				Title:       "T",
				Owner:       "bob",
				Description: "D",
			},
		}

		if err := WriteChange(root, "mymod", original, "initial"); err != nil {
			t.Fatalf("WriteChange error: %v", err)
		}

		changes, err := ReadChanges(root, "mymod")
		if err != nil {
			t.Fatalf("ReadChanges error: %v", err)
		}
		if len(changes) != 1 {
			t.Fatalf("len = %d, want 1", len(changes))
		}
		got := changes[0]
		if got.ID != original.ID {
			t.Errorf("ID = %q, want %q", got.ID, original.ID)
		}
		if got.Author != original.Author {
			t.Errorf("Author = %q, want %q", got.Author, original.Author)
		}
		if got.Summary != original.Summary {
			t.Errorf("Summary = %q, want %q", got.Summary, original.Summary)
		}
		if got.Changes.Title != original.Changes.Title {
			t.Errorf("Changes.Title = %q, want %q", got.Changes.Title, original.Changes.Title)
		}
	})
}

func TestNextSequence(t *testing.T) {
	t.Run("returns_1_empty", func(t *testing.T) {
		root := t.TempDir()
		setupModule(t, root, "mymod")

		got, err := NextSequence(root, "mymod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	})

	t.Run("returns_1_missing_dir", func(t *testing.T) {
		root := t.TempDir()

		got, err := NextSequence(root, "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	})

	t.Run("max_plus_one", func(t *testing.T) {
		root := t.TempDir()
		changesDir := setupModule(t, root, "mymod")

		writeChangeFile(t, changesDir, spec.Change{ID: "c1", Sequence: 1}, "a")
		writeChangeFile(t, changesDir, spec.Change{ID: "c2", Sequence: 2}, "b")
		writeChangeFile(t, changesDir, spec.Change{ID: "c5", Sequence: 5}, "c")

		got, err := NextSequence(root, "mymod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 6 {
			t.Errorf("got %d, want 6", got)
		}
	})
}

func TestReadSpec(t *testing.T) {
	t.Run("parses_valid_spec", func(t *testing.T) {
		root := t.TempDir()
		setupModule(t, root, "mymod")

		original := spec.SpecModule{
			ID:     "d/m",
			Domain: "d",
			Module: "m",
			Title:  "Test",
			Owner:  "alice",
		}
		writeSpecFile(t, root, "mymod", original)

		got, err := ReadSpec(root, "mymod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "d/m" {
			t.Errorf("ID = %q, want d/m", got.ID)
		}
		if got.Title != "Test" {
			t.Errorf("Title = %q, want Test", got.Title)
		}
	})

	t.Run("error_missing_file", func(t *testing.T) {
		root := t.TempDir()

		_, err := ReadSpec(root, "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSetChangeStatus(t *testing.T) {
	t.Run("updates_status_preserves_fields", func(t *testing.T) {
		root := t.TempDir()
		changesDir := setupModule(t, root, "mymod")

		ch := spec.Change{
			ID:       "chg-001",
			Sequence: 1,
			Author:   "alice",
			Summary:  "initial",
			Status:   "draft",
		}
		writeChangeFile(t, changesDir, ch, "initial")

		if err := SetChangeStatus(root, "mymod", "001_initial.yaml", "approved"); err != nil {
			t.Fatalf("SetChangeStatus error: %v", err)
		}

		// Read back
		changes, err := ReadChanges(root, "mymod")
		if err != nil {
			t.Fatalf("ReadChanges error: %v", err)
		}
		if len(changes) != 1 {
			t.Fatalf("len = %d, want 1", len(changes))
		}
		if changes[0].Status != "approved" {
			t.Errorf("Status = %q, want approved", changes[0].Status)
		}
		if changes[0].Author != "alice" {
			t.Errorf("Author = %q, want alice", changes[0].Author)
		}
		if changes[0].Summary != "initial" {
			t.Errorf("Summary = %q, want initial", changes[0].Summary)
		}
	})

	t.Run("error_nonexistent", func(t *testing.T) {
		root := t.TempDir()
		setupModule(t, root, "mymod")

		err := SetChangeStatus(root, "mymod", "999_nope.yaml", "approved")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFilterChangesByStatus(t *testing.T) {
	t.Run("filters_matching", func(t *testing.T) {
		changes := []spec.Change{
			{ID: "c1", Status: "draft"},
			{ID: "c2", Status: "approved"},
			{ID: "c3", Status: "compiled"},
		}

		got := FilterChangesByStatus(changes, "approved")
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].ID != "c2" {
			t.Errorf("ID = %q, want c2", got[0].ID)
		}
	})

	t.Run("empty_status_is_draft", func(t *testing.T) {
		changes := []spec.Change{
			{ID: "c1", Status: ""},
			{ID: "c2", Status: "approved"},
		}

		got := FilterChangesByStatus(changes, "draft")
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].ID != "c1" {
			t.Errorf("ID = %q, want c1", got[0].ID)
		}
	})
}

func TestWalkModules(t *testing.T) {
	t.Run("discovers_modules", func(t *testing.T) {
		root := t.TempDir()

		// A/changes/ - is a module
		os.MkdirAll(filepath.Join(root, "A", "changes"), 0755)
		// A/B/changes/ - is a module
		os.MkdirAll(filepath.Join(root, "A", "B", "changes"), 0755)
		// C/ - not a module (no changes/)
		os.MkdirAll(filepath.Join(root, "C"), 0755)

		got, err := WalkModules(root, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		paths := []string{got[0].Path, got[1].Path}
		if paths[0] != "A" || paths[1] != "A/B" {
			t.Errorf("paths = %v, want [A, A/B]", paths)
		}
	})

	t.Run("sorted_lexicographically", func(t *testing.T) {
		root := t.TempDir()

		os.MkdirAll(filepath.Join(root, "z-mod", "changes"), 0755)
		os.MkdirAll(filepath.Join(root, "a-mod", "changes"), 0755)
		os.MkdirAll(filepath.Join(root, "m-mod", "changes"), 0755)

		got, err := WalkModules(root, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("len = %d, want 3", len(got))
		}
		want := []string{"a-mod", "m-mod", "z-mod"}
		for i, w := range want {
			if got[i].Path != w {
				t.Errorf("got[%d].Path = %q, want %q", i, got[i].Path, w)
			}
		}
	})

	t.Run("prefix_filter", func(t *testing.T) {
		root := t.TempDir()

		os.MkdirAll(filepath.Join(root, "spec-cli", "changes"), 0755)
		os.MkdirAll(filepath.Join(root, "spec-cli", "cmd-new", "changes"), 0755)
		os.MkdirAll(filepath.Join(root, "ai-agent", "changes"), 0755)

		got, err := WalkModules(root, "spec-cli")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[0].Path != "spec-cli" {
			t.Errorf("got[0].Path = %q, want spec-cli", got[0].Path)
		}
		if got[1].Path != "spec-cli/cmd-new" {
			t.Errorf("got[1].Path = %q, want spec-cli/cmd-new", got[1].Path)
		}
	})

	t.Run("error_nonexistent_root", func(t *testing.T) {
		_, err := WalkModules("/nonexistent/path/that/does/not/exist", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
