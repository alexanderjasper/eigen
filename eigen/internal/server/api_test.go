package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/alexanderjasper/eigen/internal/spec"
	"github.com/alexanderjasper/eigen/internal/storage"
	"github.com/alexanderjasper/eigen/internal/worktree"
)

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

// testWorktreeEntry is used to set up extra worktrees in test state.
type testWorktreeEntry struct {
	name      string
	specsRoot string
	branch    string
	path      string
}

// newTestState builds a serveState for testing.
// The first specsRoot is always "main". Optional extras add additional worktrees.
func newTestState(mainSpecsRoot string, extras ...testWorktreeEntry) *serveState {
	state := &serveState{
		worktrees: make(map[string]*activeWorktree),
	}
	state.order = append(state.order, "main")
	state.worktrees["main"] = &activeWorktree{
		Entry: worktree.Entry{
			Name:   "main",
			Branch: "main",
			Path:   mainSpecsRoot,
		},
		SpecsRoot:   mainSpecsRoot,
		Branch:      "main",
		CancelWatch: func() {},
	}
	for _, e := range extras {
		p := e.path
		if p == "" {
			p = e.specsRoot
		}
		state.order = append(state.order, e.name)
		state.worktrees[e.name] = &activeWorktree{
			Entry: worktree.Entry{
				Name:   e.name,
				Branch: e.branch,
				Path:   p,
			},
			SpecsRoot:   e.specsRoot,
			Branch:      e.branch,
			CancelWatch: func() {},
		}
	}
	return state
}

// newTestMux builds a ServeMux wired identically to server.go:Start() but
// without the embed/UI/ListenAndServe. This catches routing bugs.
func newTestMux(specsRoot string) *http.ServeMux {
	state := newTestState(specsRoot)
	return newTestMuxFromState(state)
}

// newTestMuxFromState builds a ServeMux from a serveState.
func newTestMuxFromState(state *serveState) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/worktrees", worktreesHandler(state))
	mux.HandleFunc("/api/modules", modulesHandler(state))
	mux.HandleFunc("/api/modules/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/approve") {
			changeApproveHandler(state)(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/reject") {
			changeRejectHandler(state)(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/changes") {
			moduleChangesHandler(state)(w, r)
		} else {
			moduleDetailHandler(state)(w, r)
		}
	})
	return mux
}

func TestModulesHandler(t *testing.T) {
	t.Run("list_all_modules", func(t *testing.T) {
		root := t.TempDir()
		setupModule(t, root, "alpha")
		writeSpecFile(t, root, "alpha", spec.SpecModule{
			ID: "alpha", Title: "Alpha", Owner: "team-a", Status: "draft",
		})
		setupModule(t, root, "alpha/sub")
		writeSpecFile(t, root, "alpha/sub", spec.SpecModule{
			ID: "alpha/sub", Title: "Sub", Owner: "team-b", Status: "draft",
		})

		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/modules")
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		var summaries []ModuleSummary
		if err := json.NewDecoder(resp.Body).Decode(&summaries); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(summaries) != 2 {
			t.Fatalf("len = %d, want 2", len(summaries))
		}
		// Verify fields present
		for _, s := range summaries {
			if s.Path == "" {
				t.Error("Path is empty")
			}
			if s.Title == "" {
				t.Error("Title is empty")
			}
			if s.Owner == "" {
				t.Error("Owner is empty")
			}
			if s.Status == "" {
				t.Error("Status is empty")
			}
		}
	})

	t.Run("children_flag", func(t *testing.T) {
		root := t.TempDir()
		setupModule(t, root, "alpha")
		writeSpecFile(t, root, "alpha", spec.SpecModule{
			ID: "alpha", Title: "Alpha", Owner: "o", Status: "draft",
		})
		setupModule(t, root, "alpha/sub")
		writeSpecFile(t, root, "alpha/sub", spec.SpecModule{
			ID: "alpha/sub", Title: "Sub", Owner: "o", Status: "draft",
		})

		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/modules")
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		var summaries []ModuleSummary
		if err := json.NewDecoder(resp.Body).Decode(&summaries); err != nil {
			t.Fatalf("decode: %v", err)
		}

		byPath := make(map[string]ModuleSummary)
		for _, s := range summaries {
			byPath[s.Path] = s
		}

		if !byPath["alpha"].Children {
			t.Error("alpha should have children=true")
		}
		if byPath["alpha/sub"].Children {
			t.Error("alpha/sub should have children=false")
		}
	})

	t.Run("empty_root", func(t *testing.T) {
		root := t.TempDir()

		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/modules")
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var summaries []json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&summaries); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(summaries) != 0 {
			t.Errorf("len = %d, want 0", len(summaries))
		}
	})
}

func TestModuleDetailHandler(t *testing.T) {
	t.Run("returns_full_module", func(t *testing.T) {
		root := t.TempDir()
		setupModule(t, root, "alpha/sub")
		writeSpecFile(t, root, "alpha/sub", spec.SpecModule{
			ID:          "alpha/sub",
			Title:       "Sub Module",
			Owner:       "team-a",
			Status:      "draft",
			Description: "A sub module",
			Behavior:    "Does things",
			AcceptanceCriteria: []spec.AcceptanceCriterion{
				{ID: "AC-001", Description: "test criterion"},
			},
		})

		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/modules/alpha/sub")
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var m spec.SpecModule
		if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if m.Title != "Sub Module" {
			t.Errorf("Title = %q, want Sub Module", m.Title)
		}
		if m.Description != "A sub module" {
			t.Errorf("Description = %q, want A sub module", m.Description)
		}
		if m.Behavior != "Does things" {
			t.Errorf("Behavior = %q, want Does things", m.Behavior)
		}
		if len(m.AcceptanceCriteria) != 1 {
			t.Errorf("AcceptanceCriteria len = %d, want 1", len(m.AcceptanceCriteria))
		}
	})

	t.Run("not_found", func(t *testing.T) {
		root := t.TempDir()

		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/modules/nonexistent")
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 404 {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}

		var body map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if _, ok := body["error"]; !ok {
			t.Error("response missing 'error' key")
		}
	})
}

func TestModuleChangesHandler(t *testing.T) {
	t.Run("returns_changes_in_order", func(t *testing.T) {
		root := t.TempDir()
		changesDir := setupModule(t, root, "alpha")
		writeSpecFile(t, root, "alpha", spec.SpecModule{
			ID: "alpha", Title: "Alpha", Owner: "o", Status: "draft",
		})

		writeChangeFile(t, changesDir, spec.Change{
			ID: "chg-002", Sequence: 2, Summary: "second",
		}, "second")
		writeChangeFile(t, changesDir, spec.Change{
			ID: "chg-001", Sequence: 1, Summary: "first",
		}, "first")

		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/modules/alpha/changes")
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var changes []spec.Change
		if err := json.NewDecoder(resp.Body).Decode(&changes); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(changes) != 2 {
			t.Fatalf("len = %d, want 2", len(changes))
		}
		if changes[0].Sequence != 1 {
			t.Errorf("changes[0].Sequence = %d, want 1", changes[0].Sequence)
		}
		if changes[1].Sequence != 2 {
			t.Errorf("changes[1].Sequence = %d, want 2", changes[1].Sequence)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		root := t.TempDir()

		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/modules/nonexistent/changes")
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 404 {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}

		var body map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if _, ok := body["error"]; !ok {
			t.Error("response missing 'error' key")
		}
	})
}

func TestChangeApprove(t *testing.T) {
	t.Run("approves_change_file", func(t *testing.T) {
		root := t.TempDir()
		changesDir := setupModule(t, root, "test-mod")
		writeSpecFile(t, root, "test-mod", spec.SpecModule{
			ID: "test-mod", Title: "Test", Owner: "o", Status: "draft",
		})
		writeChangeFile(t, changesDir, spec.Change{
			ID: "chg-002", Sequence: 2, Summary: "feature", Status: "draft",
		}, "feature")

		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		resp, err := http.Post(ts.URL+"/api/modules/test-mod/changes/002_feature.yaml/approve", "application/json", nil)
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var respBody map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if respBody["status"] != "approved" {
			t.Errorf("status = %q, want approved", respBody["status"])
		}

		// Verify on disk
		changes, err := storage.ReadChanges(root, "test-mod")
		if err != nil {
			t.Fatalf("ReadChanges error: %v", err)
		}
		if len(changes) != 1 {
			t.Fatalf("len = %d, want 1", len(changes))
		}
		if changes[0].Status != "approved" {
			t.Errorf("disk status = %q, want approved", changes[0].Status)
		}
	})

	t.Run("404_missing_file", func(t *testing.T) {
		root := t.TempDir()
		setupModule(t, root, "test-mod")
		writeSpecFile(t, root, "test-mod", spec.SpecModule{
			ID: "test-mod", Title: "Test", Owner: "o", Status: "draft",
		})

		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		resp, err := http.Post(ts.URL+"/api/modules/test-mod/changes/999_missing.yaml/approve", "application/json", nil)
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 404 {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}

		var respBody map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if _, ok := respBody["error"]; !ok {
			t.Error("response missing 'error' key")
		}
	})
}

func TestChangeReject(t *testing.T) {
	t.Run("rejects_with_comment", func(t *testing.T) {
		root := t.TempDir()
		changesDir := setupModule(t, root, "test-mod")
		writeSpecFile(t, root, "test-mod", spec.SpecModule{
			ID: "test-mod", Title: "Test", Owner: "o", Status: "draft",
		})
		writeChangeFile(t, changesDir, spec.Change{
			ID: "chg-002", Sequence: 2, Summary: "feature", Status: "draft",
		}, "feature")

		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		reqBody, _ := json.Marshal(map[string]string{"comment": "needs more detail on edge cases"})
		resp, err := http.Post(ts.URL+"/api/modules/test-mod/changes/002_feature.yaml/reject", "application/json", bytes.NewReader(reqBody))
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var respBody map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if respBody["status"] != "draft" {
			t.Errorf("status = %q, want draft", respBody["status"])
		}
		if respBody["review_comment"] != "needs more detail on edge cases" {
			t.Errorf("review_comment = %q, want 'needs more detail on edge cases'", respBody["review_comment"])
		}

		// Verify on disk
		changes, err := storage.ReadChanges(root, "test-mod")
		if err != nil {
			t.Fatalf("ReadChanges error: %v", err)
		}
		if len(changes) != 1 {
			t.Fatalf("len = %d, want 1", len(changes))
		}
		if changes[0].ReviewComment != "needs more detail on edge cases" {
			t.Errorf("disk ReviewComment = %q", changes[0].ReviewComment)
		}
		if changes[0].Status != "draft" {
			t.Errorf("disk Status = %q, want draft", changes[0].Status)
		}
	})

	t.Run("400_empty_comment", func(t *testing.T) {
		root := t.TempDir()
		changesDir := setupModule(t, root, "test-mod")
		writeSpecFile(t, root, "test-mod", spec.SpecModule{
			ID: "test-mod", Title: "Test", Owner: "o", Status: "draft",
		})
		writeChangeFile(t, changesDir, spec.Change{
			ID: "chg-002", Sequence: 2, Summary: "feature", Status: "draft",
		}, "feature")

		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		reqBody, _ := json.Marshal(map[string]string{"comment": ""})
		resp, err := http.Post(ts.URL+"/api/modules/test-mod/changes/002_feature.yaml/reject", "application/json", bytes.NewReader(reqBody))
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Fatalf("status = %d, want 400", resp.StatusCode)
		}

		var respBody map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if _, ok := respBody["error"]; !ok {
			t.Error("response missing 'error' key")
		}
	})

	t.Run("404_missing_file", func(t *testing.T) {
		root := t.TempDir()
		setupModule(t, root, "test-mod")
		writeSpecFile(t, root, "test-mod", spec.SpecModule{
			ID: "test-mod", Title: "Test", Owner: "o", Status: "draft",
		})

		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		reqBody, _ := json.Marshal(map[string]string{"comment": "some feedback"})
		resp, err := http.Post(ts.URL+"/api/modules/test-mod/changes/999_missing.yaml/reject", "application/json", bytes.NewReader(reqBody))
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 404 {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}

		var respBody map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if _, ok := respBody["error"]; !ok {
			t.Error("response missing 'error' key")
		}
	})
}

func TestChangesEndpointIncludesReviewComment(t *testing.T) {
	root := t.TempDir()
	changesDir := setupModule(t, root, "test-mod")
	writeSpecFile(t, root, "test-mod", spec.SpecModule{
		ID: "test-mod", Title: "Test", Owner: "o", Status: "draft",
	})
	writeChangeFile(t, changesDir, spec.Change{
		ID: "chg-002", Sequence: 2, Summary: "feature", Status: "draft",
	}, "feature")

	// Set a review comment via storage
	if err := storage.SetChangeComment(root, "test-mod", "002_feature.yaml", "needs edge cases"); err != nil {
		t.Fatalf("SetChangeComment error: %v", err)
	}

	ts := httptest.NewServer(newTestMux(root))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/modules/test-mod/changes")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var changes []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&changes); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("len = %d, want 1", len(changes))
	}
	rc, ok := changes[0]["review_comment"]
	if !ok {
		t.Fatal("response missing review_comment field")
	}
	if rc != "needs edge cases" {
		t.Errorf("review_comment = %q, want 'needs edge cases'", rc)
	}
}

// ── Worktree-aware tests ───────────────────────────────────────────────────────

func TestModulesHandlerWorktreeFields(t *testing.T) {
	root := t.TempDir()
	setupModule(t, root, "alpha")
	writeSpecFile(t, root, "alpha", spec.SpecModule{
		ID: "alpha", Title: "Alpha", Owner: "o", Status: "draft",
	})

	state := newTestState(root)
	ts := httptest.NewServer(newTestMuxFromState(state))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/modules")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	var summaries []ModuleSummary
	if err := json.NewDecoder(resp.Body).Decode(&summaries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	s := summaries[0]
	if s.Worktree == "" {
		t.Error("Worktree field is empty")
	}
	if s.Branch == "" {
		t.Error("Branch field is empty")
	}
	if s.Worktree != "main" {
		t.Errorf("Worktree = %q, want main", s.Worktree)
	}
}

func TestModulesHandlerMultiWorktree(t *testing.T) {
	mainRoot := t.TempDir()
	setupModule(t, mainRoot, "spec-cli")
	writeSpecFile(t, mainRoot, "spec-cli", spec.SpecModule{
		ID: "spec-cli", Title: "Spec CLI", Owner: "o", Status: "draft",
	})

	wtRoot := t.TempDir()
	setupModule(t, wtRoot, "spec-cli")
	writeSpecFile(t, wtRoot, "spec-cli", spec.SpecModule{
		ID: "spec-cli", Title: "Spec CLI WT", Owner: "o", Status: "draft",
	})
	setupModule(t, wtRoot, "infra")
	writeSpecFile(t, wtRoot, "infra", spec.SpecModule{
		ID: "infra", Title: "Infra", Owner: "o", Status: "draft",
	})

	state := newTestState(mainRoot, testWorktreeEntry{
		name:      "feature-foo",
		specsRoot: wtRoot,
		branch:    "feature/foo",
	})

	ts := httptest.NewServer(newTestMuxFromState(state))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/modules")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	var summaries []ModuleSummary
	if err := json.NewDecoder(resp.Body).Decode(&summaries); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Expect 2: spec-cli from main (no unique wt changes), infra from feature-foo (new in wt).
	// spec-cli is NOT duplicated in the worktree group because it has no unique changes there.
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	// Check that every summary has worktree and branch.
	for _, s := range summaries {
		if s.Worktree == "" {
			t.Errorf("path %q: Worktree is empty", s.Path)
		}
		if s.Branch == "" {
			t.Errorf("path %q: Branch is empty", s.Path)
		}
	}

	// Check that feature-foo summaries have the right branch.
	for _, s := range summaries {
		if s.Worktree == "feature-foo" && s.Branch != "feature/foo" {
			t.Errorf("path %q: Branch = %q, want feature/foo", s.Path, s.Branch)
		}
	}
}

func TestWorktreesHandler(t *testing.T) {
	mainRoot := t.TempDir()
	wtRoot := t.TempDir()

	state := newTestState(mainRoot, testWorktreeEntry{
		name:      "feature-foo",
		specsRoot: wtRoot,
		branch:    "feature/foo",
		path:      wtRoot, // exists on disk
	})

	// Add an orphaned worktree whose path doesn't exist.
	orphanPath := t.TempDir() + "/nonexistent"
	state.order = append(state.order, "orphan")
	state.worktrees["orphan"] = &activeWorktree{
		Entry: worktree.Entry{
			Name:   "orphan",
			Branch: "orphan",
			Path:   orphanPath,
		},
		SpecsRoot:   orphanPath + "/specs",
		Branch:      "orphan",
		CancelWatch: func() {},
	}

	ts := httptest.NewServer(newTestMuxFromState(state))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/worktrees")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var infos []WorktreeInfo
	if err := json.NewDecoder(resp.Body).Decode(&infos); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(infos) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(infos))
	}
	if infos[0].Name != "main" {
		t.Errorf("infos[0].Name = %q, want main", infos[0].Name)
	}

	byName := make(map[string]WorktreeInfo)
	for _, info := range infos {
		byName[info.Name] = info
	}

	if byName["feature-foo"].Status != "active" {
		t.Errorf("feature-foo status = %q, want active", byName["feature-foo"].Status)
	}
	if byName["orphan"].Status != "orphaned" {
		t.Errorf("orphan status = %q, want orphaned", byName["orphan"].Status)
	}
}

func TestModuleDetailDisambiguate409(t *testing.T) {
	mainRoot := t.TempDir()
	setupModule(t, mainRoot, "spec-cli")
	writeSpecFile(t, mainRoot, "spec-cli", spec.SpecModule{
		ID: "spec-cli", Title: "Spec CLI", Owner: "o", Status: "draft",
	})

	wtRoot := t.TempDir()
	setupModule(t, wtRoot, "spec-cli")
	writeSpecFile(t, wtRoot, "spec-cli", spec.SpecModule{
		ID: "spec-cli", Title: "Spec CLI WT", Owner: "o", Status: "draft",
	})

	state := newTestState(mainRoot, testWorktreeEntry{
		name:      "feature-foo",
		specsRoot: wtRoot,
		branch:    "feature/foo",
	})

	ts := httptest.NewServer(newTestMuxFromState(state))
	defer ts.Close()

	// Same path in two worktrees, no ?worktree param — should return 409.
	resp, err := http.Get(ts.URL + "/api/modules/spec-cli")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 409 {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Error("response missing 'error' key")
	}
	if _, ok := body["worktrees"]; !ok {
		t.Error("response missing 'worktrees' key")
	}
}

func TestModuleDetailDisambiguateParam(t *testing.T) {
	mainRoot := t.TempDir()
	setupModule(t, mainRoot, "spec-cli")
	writeSpecFile(t, mainRoot, "spec-cli", spec.SpecModule{
		ID: "spec-cli", Title: "Main Spec CLI", Owner: "o", Status: "draft",
	})

	wtRoot := t.TempDir()
	setupModule(t, wtRoot, "spec-cli")
	writeSpecFile(t, wtRoot, "spec-cli", spec.SpecModule{
		ID: "spec-cli", Title: "WT Spec CLI", Owner: "o", Status: "draft",
	})

	state := newTestState(mainRoot, testWorktreeEntry{
		name:      "feature-foo",
		specsRoot: wtRoot,
		branch:    "feature/foo",
	})

	ts := httptest.NewServer(newTestMuxFromState(state))
	defer ts.Close()

	// With ?worktree=main — should return 200 for the main copy.
	resp, err := http.Get(ts.URL + "/api/modules/spec-cli?worktree=main")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var m spec.SpecModule
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if m.Title != "Main Spec CLI" {
		t.Errorf("Title = %q, want 'Main Spec CLI'", m.Title)
	}

	// With ?worktree=feature-foo — should return 200 for the worktree copy.
	resp2, err := http.Get(ts.URL + "/api/modules/spec-cli?worktree=feature-foo")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp2.StatusCode)
	}

	var m2 spec.SpecModule
	if err := json.NewDecoder(resp2.Body).Decode(&m2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if m2.Title != "WT Spec CLI" {
		t.Errorf("Title = %q, want 'WT Spec CLI'", m2.Title)
	}
}
