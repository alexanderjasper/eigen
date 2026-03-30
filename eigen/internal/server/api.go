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
	Path     string `json:"path"`
	Title    string `json:"title"`
	Owner    string `json:"owner"`
	Status   string `json:"status"`
	Children bool   `json:"children"`
}

func modulesHandler(specsRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		refs, err := storage.WalkModules(specsRoot, "")
		if err != nil {
			jsonError(w, "failed to list modules: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Build a set of all paths for children detection.
		pathSet := make(map[string]bool, len(refs))
		for _, ref := range refs {
			pathSet[ref.Path] = true
		}

		summaries := make([]ModuleSummary, 0, len(refs))
		for _, ref := range refs {
			s, err := storage.ReadSpec(specsRoot, ref.Path)
			if err != nil {
				// Include with empty fields rather than failing the whole list.
				summaries = append(summaries, ModuleSummary{Path: ref.Path})
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
			summaries = append(summaries, ModuleSummary{
				Path:     ref.Path,
				Title:    s.Title,
				Owner:    s.Owner,
				Status:   s.Status,
				Children: hasChildren,
			})
		}

		writeJSON(w, summaries)
	}
}

func moduleDetailHandler(specsRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := modulePath(r)
		s, err := storage.ReadSpec(specsRoot, path)
		if err != nil {
			jsonError(w, "module not found: "+err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, s)
	}
}

func moduleChangesHandler(specsRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := modulePath(r)
		changes, err := storage.ReadChanges(specsRoot, path)
		if err != nil {
			jsonError(w, "changes not found: "+err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, changes)
	}
}

// modulePath extracts the module path from /api/modules/<path> or /api/modules/<path>/changes.
func modulePath(r *http.Request) string {
	// Strip leading /api/modules/
	p := strings.TrimPrefix(r.URL.Path, "/api/modules/")
	// Strip trailing /changes
	p = strings.TrimSuffix(p, "/changes")
	return p
}

// parseChangeActionPath extracts modulePath and filename from
// /api/modules/<module-path>/changes/<filename>/(approve|reject).
func parseChangeActionPath(urlPath string) (modPath, filename string) {
	// Strip /api/modules/ prefix
	p := strings.TrimPrefix(urlPath, "/api/modules/")
	// Strip trailing /approve or /reject
	p = strings.TrimSuffix(p, "/approve")
	p = strings.TrimSuffix(p, "/reject")
	// Now p = <module-path>/changes/<filename>
	// Split on /changes/ to get module path and filename
	idx := strings.LastIndex(p, "/changes/")
	if idx < 0 {
		return "", ""
	}
	return p[:idx], p[idx+len("/changes/"):]
}

func changeApproveHandler(specsRoot string) http.HandlerFunc {
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

func changeRejectHandler(specsRoot string) http.HandlerFunc {
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

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
