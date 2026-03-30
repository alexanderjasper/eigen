package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/alexanderjasper/eigen/internal/storage"
)

// ModuleSummary is the JSON shape returned by GET /api/modules.
type ModuleSummary struct {
	Path              string `json:"path"`
	Title             string `json:"title"`
	Owner             string `json:"owner"`
	Status            string `json:"status"`
	DeprecationReason string `json:"deprecation_reason,omitempty"`
	Children          bool   `json:"children"`
	Worktree          string `json:"worktree"`
	Branch            string `json:"branch"`
}

// WorktreeInfo is the JSON shape returned by GET /api/worktrees.
type WorktreeInfo struct {
	Name   string `json:"name"`
	Branch string `json:"branch"`
	Path   string `json:"path"`
	Status string `json:"status"` // "active" | "orphaned"
}

func modulesHandler(state *serveState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state.mu.RLock()
		order := make([]string, len(state.order))
		copy(order, state.order)
		// Snapshot active worktrees.
		type wtSnap struct {
			specsRoot string
			branch    string
			name      string
		}
		snaps := make([]wtSnap, 0, len(order))
		for _, name := range order {
			aw, ok := state.worktrees[name]
			if !ok {
				continue
			}
			snaps = append(snaps, wtSnap{
				specsRoot: aw.SpecsRoot,
				branch:    aw.Branch,
				name:      name,
			})
		}
		state.mu.RUnlock()

		// Build set of module paths "claimed" by a non-main worktree (unique WIP).
		// These will be hidden from the main group so each module appears only once.
		var mainSpecsRoot string
		for _, name := range order {
			if name == "main" {
				if aw, ok := state.worktrees[name]; ok {
					mainSpecsRoot = aw.SpecsRoot
				}
				break
			}
		}
		claimedByWorktree := make(map[string]bool)
		if mainSpecsRoot != "" {
			for _, snap := range snaps {
				if snap.name == "main" {
					continue
				}
				wtRefs, err := storage.WalkModules(snap.specsRoot, "")
				if err != nil {
					continue
				}
				for _, ref := range wtRefs {
					if hasUniqueChanges(snap.specsRoot, mainSpecsRoot, ref.Path) {
						claimedByWorktree[ref.Path] = true
					}
				}
			}
		}

		var allSummaries []ModuleSummary
		for _, snap := range snaps {
			refs, err := storage.WalkModules(snap.specsRoot, "")
			if err != nil {
				continue
			}

			// Build a set of all paths for children detection within this worktree.
			pathSet := make(map[string]bool, len(refs))
			for _, ref := range refs {
				pathSet[ref.Path] = true
			}

		for _, ref := range refs {
				// For main: skip modules claimed by a worktree (they appear there instead).
				// For non-main: skip modules with no unique changes vs main.
				if snap.name == "main" {
					if claimedByWorktree[ref.Path] {
						continue
					}
				} else if mainSpecsRoot != "" && !hasUniqueChanges(snap.specsRoot, mainSpecsRoot, ref.Path) {
					continue
				}
				s, err := storage.ReadSpec(snap.specsRoot, ref.Path)
				if err != nil {
					allSummaries = append(allSummaries, ModuleSummary{
						Path:     ref.Path,
						Worktree: snap.name,
						Branch:   snap.branch,
					})
					continue
				}
				if s.Status == "removed" {
					continue
				}
				hasChildren := false
				prefix := ref.Path + "/"
				for p := range pathSet {
					if strings.HasPrefix(p, prefix) {
						hasChildren = true
						break
					}
				}
				allSummaries = append(allSummaries, ModuleSummary{
					Path:              ref.Path,
					Title:             s.Title,
					Owner:             s.Owner,
					Status:            s.Status,
					DeprecationReason: s.DeprecationReason,
					Children:          hasChildren,
					Worktree:          snap.name,
					Branch:            snap.branch,
				})
			}
		}

		if allSummaries == nil {
			allSummaries = []ModuleSummary{}
		}
		writeJSON(w, allSummaries)
	}
}

func moduleDetailHandler(state *serveState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := modulePath(r)
		wt := r.URL.Query().Get("worktree")

		specsRoot, err := resolveSpecsRoot(state, path, wt)
		if err != nil {
			if isAmbiguous(err) {
				writeJSON409(w, ambiguousErr(err))
				return
			}
			jsonError(w, "module not found: "+err.Error(), http.StatusNotFound)
			return
		}

		s, err := storage.ReadSpec(specsRoot, path)
		if err != nil {
			jsonError(w, "module not found: "+err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, s)
	}
}

func moduleChangesHandler(state *serveState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := modulePathForChanges(r)
		wt := r.URL.Query().Get("worktree")

		specsRoot, err := resolveSpecsRoot(state, path, wt)
		if err != nil {
			if isAmbiguous(err) {
				writeJSON409(w, ambiguousErr(err))
				return
			}
			jsonError(w, "changes not found: "+err.Error(), http.StatusNotFound)
			return
		}

		changes, err := storage.ReadChanges(specsRoot, path)
		if err != nil {
			jsonError(w, "changes not found: "+err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, changes)
	}
}

func worktreesHandler(state *serveState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state.mu.RLock()
		order := make([]string, len(state.order))
		copy(order, state.order)
		type wtSnap struct {
			name   string
			branch string
			path   string
		}
		snaps := make([]wtSnap, 0, len(order))
		for _, name := range order {
			aw, ok := state.worktrees[name]
			if !ok {
				continue
			}
			snaps = append(snaps, wtSnap{
				name:   name,
				branch: aw.Branch,
				path:   aw.Entry.Path,
			})
		}
		state.mu.RUnlock()

		infos := make([]WorktreeInfo, 0, len(snaps))
		for _, snap := range snaps {
			status := "active"
			if snap.name != "main" {
				if _, err := os.Stat(snap.path); os.IsNotExist(err) {
					status = "orphaned"
				}
			}
			infos = append(infos, WorktreeInfo{
				Name:   snap.name,
				Branch: snap.branch,
				Path:   snap.path,
				Status: status,
			})
		}
		writeJSON(w, infos)
	}
}

// GlobalChangeEntry is the JSON shape returned by GET /api/changes.
type GlobalChangeEntry struct {
	ModulePath string `json:"module_path"`
	Worktree   string `json:"worktree"`
	// Embed all Change fields inline.
	Format          string            `json:"format,omitempty"`
	ID              string            `json:"id"`
	Sequence        int               `json:"sequence"`
	Timestamp       string            `json:"timestamp"`
	Author          string            `json:"author"`
	Type            string            `json:"type"`
	Summary         string            `json:"summary"`
	Reason          string            `json:"reason"`
	Status          string            `json:"status,omitempty"`
	ReviewComment   string            `json:"review_comment,omitempty"`
	CompiledCommits []string          `json:"compiled_commits,omitempty"`
	Filename        string            `json:"filename,omitempty"`
	Changes         interface{}       `json:"changes,omitempty"`
}

// globalChangesHandler handles GET /api/changes?status=draft
// Returns all changes across all worktrees and modules, optionally filtered by status.
func globalChangesHandler(state *serveState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statusFilter := r.URL.Query().Get("status")

		state.mu.RLock()
		order := make([]string, len(state.order))
		copy(order, state.order)
		type wtSnap struct {
			name      string
			specsRoot string
		}
		snaps := make([]wtSnap, 0, len(order))
		for _, name := range order {
			aw, ok := state.worktrees[name]
			if !ok {
				continue
			}
			snaps = append(snaps, wtSnap{name: name, specsRoot: aw.SpecsRoot})
		}
		state.mu.RUnlock()

		var results []GlobalChangeEntry
		for _, snap := range snaps {
			refs, err := storage.WalkModules(snap.specsRoot, "")
			if err != nil {
				continue
			}
			for _, ref := range refs {
				changes, err := storage.ReadChanges(snap.specsRoot, ref.Path)
				if err != nil {
					continue
				}
				for _, ch := range changes {
					if statusFilter != "" {
						chStatus := ch.Status
						if chStatus == "" {
							chStatus = "draft"
						}
						if chStatus != statusFilter {
							continue
						}
					}
					results = append(results, GlobalChangeEntry{
						ModulePath:      ref.Path,
						Worktree:        snap.name,
						Format:          ch.Format,
						ID:              ch.ID,
						Sequence:        ch.Sequence,
						Timestamp:       ch.Timestamp,
						Author:          ch.Author,
						Type:            ch.Type,
						Summary:         ch.Summary,
						Reason:          ch.Reason,
						Status:          ch.Status,
						ReviewComment:   ch.ReviewComment,
						CompiledCommits: ch.CompiledCommits,
						Filename:        ch.Filename,
						Changes:         ch.Changes,
					})
				}
			}
		}

		if results == nil {
			results = []GlobalChangeEntry{}
		}
		writeJSON(w, results)
	}
}

// changeApproveHandler handles POST /api/modules/<path>/changes/<filename>/approve.
func changeApproveHandler(state *serveState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		modPath, filename := parseChangeActionPath(r.URL.Path)
		if modPath == "" || filename == "" {
			jsonError(w, "invalid path", http.StatusBadRequest)
			return
		}
		wt := r.URL.Query().Get("worktree")
		specsRoot, err := resolveSpecsRoot(state, modPath, wt)
		if err != nil {
			if isAmbiguous(err) {
				writeJSON409(w, ambiguousErr(err))
				return
			}
			jsonError(w, "module not found", http.StatusNotFound)
			return
		}

		// Optionally accept a comment in the request body (backward-compatible).
		var req struct {
			Comment string `json:"comment"`
		}
		if r.Body != nil && r.ContentLength != 0 {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		if strings.TrimSpace(req.Comment) != "" {
			if err := storage.SetChangeComment(specsRoot, modPath, filename, req.Comment); err != nil && !errors.Is(err, os.ErrNotExist) {
				jsonError(w, "failed to set comment: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if err := storage.SetChangeStatus(specsRoot, modPath, filename, "approved", nil); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				jsonError(w, "change file not found", http.StatusNotFound)
				return
			}
			jsonError(w, "failed to approve: "+err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]string{"status": "approved"})
	}
}

// changeRejectHandler handles POST /api/modules/<path>/changes/<filename>/reject.
func changeRejectHandler(state *serveState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		modPath, filename := parseChangeActionPath(r.URL.Path)
		if modPath == "" || filename == "" {
			jsonError(w, "invalid path", http.StatusBadRequest)
			return
		}
		wt := r.URL.Query().Get("worktree")
		specsRoot, err := resolveSpecsRoot(state, modPath, wt)
		if err != nil {
			if isAmbiguous(err) {
				writeJSON409(w, ambiguousErr(err))
				return
			}
			jsonError(w, "module not found", http.StatusNotFound)
			return
		}

		var req struct {
			Comment string `json:"comment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Comment) == "" {
			jsonError(w, "comment is required", http.StatusBadRequest)
			return
		}

		if err := storage.SetChangeComment(specsRoot, modPath, filename, req.Comment); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				jsonError(w, "change file not found", http.StatusNotFound)
				return
			}
			jsonError(w, "failed to reject: "+err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]string{"status": "draft", "review_comment": req.Comment})
	}
}

// ── Path helpers ──────────────────────────────────────────────────────────────

// modulePath extracts the module path from /api/modules/<path>.
func modulePath(r *http.Request) string {
	return strings.TrimPrefix(r.URL.Path, "/api/modules/")
}

// modulePathForChanges extracts the module path from /api/modules/<path>/changes.
func modulePathForChanges(r *http.Request) string {
	p := strings.TrimPrefix(r.URL.Path, "/api/modules/")
	return strings.TrimSuffix(p, "/changes")
}

// parseChangeActionPath extracts modulePath and filename from
// /api/modules/<module-path>/changes/<filename>/(approve|reject).
func parseChangeActionPath(urlPath string) (modPath, filename string) {
	p := strings.TrimPrefix(urlPath, "/api/modules/")
	p = strings.TrimSuffix(p, "/approve")
	p = strings.TrimSuffix(p, "/reject")
	idx := strings.LastIndex(p, "/changes/")
	if idx < 0 {
		return "", ""
	}
	return p[:idx], p[idx+len("/changes/"):]
}

// ── Resolution helpers ────────────────────────────────────────────────────────

// ambiguityError is returned when a path exists in multiple worktrees and no ?worktree= was given.
type ambiguityError struct {
	worktrees []string
}

func (e *ambiguityError) Error() string {
	return "ambiguous module path: exists in multiple worktrees: " + strings.Join(e.worktrees, ", ")
}

func isAmbiguous(err error) bool {
	var ae *ambiguityError
	return errors.As(err, &ae)
}

func ambiguousErr(err error) map[string]interface{} {
	var ae *ambiguityError
	if errors.As(err, &ae) {
		return map[string]interface{}{
			"error":     err.Error(),
			"worktrees": ae.worktrees,
		}
	}
	return map[string]interface{}{"error": err.Error()}
}

// resolveSpecsRoot finds the specsRoot for the given module path.
// If wt is specified, it resolves to that worktree's specsRoot directly.
// If wt is empty, it finds which worktrees contain the module.
// Returns an ambiguityError if the path is ambiguous and wt is empty.
func resolveSpecsRoot(state *serveState, modPath, wt string) (string, error) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	if wt != "" {
		aw, ok := state.worktrees[wt]
		if !ok {
			return "", errors.New("worktree not found: " + wt)
		}
		return aw.SpecsRoot, nil
	}

	// Find all worktrees that contain this module.
	var matches []string
	var matchedSpecsRoot string
	for _, name := range state.order {
		aw, ok := state.worktrees[name]
		if !ok {
			continue
		}
		if moduleExists(aw.SpecsRoot, modPath) {
			matches = append(matches, name)
			matchedSpecsRoot = aw.SpecsRoot
		}
	}

	switch len(matches) {
	case 0:
		return "", errors.New("module not found: " + modPath)
	case 1:
		return matchedSpecsRoot, nil
	default:
		return "", &ambiguityError{worktrees: matches}
	}
}

// moduleExists checks if a module at path exists in the given specsRoot.
func moduleExists(specsRoot, modPath string) bool {
	refs, err := storage.WalkModules(specsRoot, modPath)
	if err != nil {
		return false
	}
	for _, ref := range refs {
		if ref.Path == modPath {
			return true
		}
	}
	return false
}

// hasUniqueChanges reports whether the module at modPath in wtSpecsRoot has at
// least one change file that does not exist in mainSpecsRoot. Used to decide
// whether a module is "interesting" in a non-main worktree — i.e. it has work
// in progress that isn't in main yet.
func hasUniqueChanges(wtSpecsRoot, mainSpecsRoot, modPath string) bool {
	mainChanges, mainErr := storage.ReadChanges(mainSpecsRoot, modPath)
	if mainErr != nil {
		return true // module doesn't exist in main — it's new in this worktree
	}
	mainFiles := make(map[string]bool, len(mainChanges))
	for _, ch := range mainChanges {
		mainFiles[ch.Filename] = true
	}
	wtChanges, _ := storage.ReadChanges(wtSpecsRoot, modPath)
	for _, ch := range wtChanges {
		if !mainFiles[ch.Filename] {
			return true
		}
	}
	return false
}

// ── JSON helpers ──────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeJSON409(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
