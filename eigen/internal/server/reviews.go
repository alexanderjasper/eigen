package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// ChangeEntry represents a single change file in a batch review session.
type ChangeEntry struct {
	ChangeID   string `json:"change_id"`
	FilePath   string `json:"file_path"`
	ChangeYAML string `json:"change_yaml"`
}

// ReviewSession holds the state of a batch review.
type ReviewSession struct {
	SessionID      string            `json:"session_id"`
	ModulePath     string            `json:"module_path"`
	Changes        []ChangeEntry     `json:"changes"`
	Status         string            `json:"status"`   // "pending" | "submitted"
	Decision       string            `json:"decision"` // "" | "approved" | "rejected"
	ChangeComments map[string]string `json:"change_comments"` // keyed by change_id
}

var (
	reviewMu    sync.Mutex
	reviewStore = make(map[string]*ReviewSession)
)

func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// createReviewHandler handles POST /api/reviews.
func createReviewHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			ModulePath string        `json:"module_path"`
			Changes    []ChangeEntry `json:"changes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		sessionID, err := generateSessionID()
		if err != nil {
			jsonError(w, "failed to generate session id: "+err.Error(), http.StatusInternalServerError)
			return
		}

		session := &ReviewSession{
			SessionID:      sessionID,
			ModulePath:     req.ModulePath,
			Changes:        req.Changes,
			Status:         "pending",
			Decision:       "",
			ChangeComments: make(map[string]string),
		}

		reviewMu.Lock()
		reviewStore[sessionID] = session
		reviewMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"session_id": sessionID})
	}
}

// getReviewHandler handles GET /api/reviews/<session-id>.
func getReviewHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract session ID: strip /api/reviews/
		id := strings.TrimPrefix(r.URL.Path, "/api/reviews/")

		reviewMu.Lock()
		session, ok := reviewStore[id]
		reviewMu.Unlock()

		if !ok {
			jsonError(w, "review session not found", http.StatusNotFound)
			return
		}

		writeJSON(w, session)
	}
}

// submitReviewHandler handles POST /api/reviews/<session-id>/submit.
func submitReviewHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract session ID: strip /api/reviews/ and /submit
		path := strings.TrimPrefix(r.URL.Path, "/api/reviews/")
		id := strings.TrimSuffix(path, "/submit")

		var req struct {
			Decision       string            `json:"decision"`
			ChangeComments map[string]string `json:"change_comments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		reviewMu.Lock()
		session, ok := reviewStore[id]
		if ok {
			session.Status = "submitted"
			session.Decision = req.Decision
			if req.ChangeComments != nil {
				session.ChangeComments = req.ChangeComments
			}
		}
		reviewMu.Unlock()

		if !ok {
			jsonError(w, "review session not found", http.StatusNotFound)
			return
		}

		writeJSON(w, map[string]string{"session_id": id})
	}
}

// pendingReviewHandler handles GET /api/reviews/pending.
func pendingReviewHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		reviewMu.Lock()
		var pending *ReviewSession
		for _, s := range reviewStore {
			if s.Status == "pending" {
				pending = s
				break
			}
		}
		reviewMu.Unlock()

		if pending == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		writeJSON(w, map[string]string{"session_id": pending.SessionID})
	}
}
