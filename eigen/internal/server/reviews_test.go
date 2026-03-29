package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// resetReviewStore clears the package-level reviewStore between tests.
func resetReviewStore() {
	reviewMu.Lock()
	reviewStore = make(map[string]*ReviewSession)
	reviewMu.Unlock()
}

// createTestSession posts a review session and returns the session_id.
func createTestSession(t *testing.T, tsURL string, modulePath string, changes []ChangeEntry) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"module_path": modulePath,
		"changes":     changes,
	})
	resp, err := http.Post(tsURL+"/api/reviews", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create review request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("create review status = %d, want 201", resp.StatusCode)
	}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id, ok := result["session_id"]
	if !ok || id == "" {
		t.Fatal("create response missing session_id")
	}
	return id
}

func TestCreateReview(t *testing.T) {
	t.Run("creates_pending_session", func(t *testing.T) {
		resetReviewStore()
		root := t.TempDir()
		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		changes := []ChangeEntry{
			{ChangeID: "chg-001", FilePath: "changes/001_init.yaml", ChangeYAML: "id: chg-001"},
		}
		sessionID := createTestSession(t, ts.URL, "my/module", changes)

		// Verify session_id is a UUID-like string
		if len(sessionID) == 0 {
			t.Fatal("session_id is empty")
		}

		// GET the session to confirm it is pending
		resp, err := http.Get(ts.URL + "/api/reviews/" + sessionID)
		if err != nil {
			t.Fatalf("get review error: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("get review status = %d, want 200", resp.StatusCode)
		}
		var session ReviewSession
		if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
			t.Fatalf("decode session: %v", err)
		}
		if session.Status != "pending" {
			t.Errorf("status = %q, want pending", session.Status)
		}
	})

	t.Run("invalid_body", func(t *testing.T) {
		resetReviewStore()
		root := t.TempDir()
		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		resp, err := http.Post(ts.URL+"/api/reviews", "application/json", strings.NewReader("{invalid"))
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Fatalf("status = %d, want 400", resp.StatusCode)
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

func TestGetPendingReview(t *testing.T) {
	t.Run("returns_pending", func(t *testing.T) {
		resetReviewStore()
		root := t.TempDir()
		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		changes := []ChangeEntry{
			{ChangeID: "chg-001", FilePath: "changes/001_init.yaml", ChangeYAML: "id: chg-001"},
		}
		sessionID := createTestSession(t, ts.URL, "my/module", changes)

		resp, err := http.Get(ts.URL + "/api/reviews/pending")
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var body map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["session_id"] != sessionID {
			t.Errorf("session_id = %q, want %q", body["session_id"], sessionID)
		}
	})

	t.Run("no_pending_returns_204", func(t *testing.T) {
		resetReviewStore()
		root := t.TempDir()
		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/reviews/pending")
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 204 {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
	})

	t.Run("submitted_not_returned", func(t *testing.T) {
		resetReviewStore()
		root := t.TempDir()
		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		changes := []ChangeEntry{
			{ChangeID: "chg-001", FilePath: "changes/001_init.yaml", ChangeYAML: "id: chg-001"},
		}
		sessionID := createTestSession(t, ts.URL, "my/module", changes)

		// Submit the session
		submitBody, _ := json.Marshal(map[string]any{
			"decision":        "approved",
			"change_comments": map[string]string{},
		})
		resp, err := http.Post(ts.URL+"/api/reviews/"+sessionID+"/submit", "application/json", bytes.NewReader(submitBody))
		if err != nil {
			t.Fatalf("submit error: %v", err)
		}
		resp.Body.Close()

		// Now pending should return 204
		resp, err = http.Get(ts.URL + "/api/reviews/pending")
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 204 {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
	})
}

func TestGetReview(t *testing.T) {
	t.Run("returns_full_session", func(t *testing.T) {
		resetReviewStore()
		root := t.TempDir()
		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		changes := []ChangeEntry{
			{ChangeID: "chg-001", FilePath: "changes/001_init.yaml", ChangeYAML: "id: chg-001"},
		}
		sessionID := createTestSession(t, ts.URL, "my/module", changes)

		resp, err := http.Get(ts.URL + "/api/reviews/" + sessionID)
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var raw map[string]json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			t.Fatalf("decode: %v", err)
		}

		requiredFields := []string{"session_id", "module_path", "changes", "status", "decision", "change_comments"}
		for _, f := range requiredFields {
			if _, ok := raw[f]; !ok {
				t.Errorf("response missing field %q", f)
			}
		}
	})

	t.Run("new_session_pending", func(t *testing.T) {
		resetReviewStore()
		root := t.TempDir()
		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		changes := []ChangeEntry{
			{ChangeID: "chg-001", FilePath: "changes/001_init.yaml", ChangeYAML: "id: chg-001"},
		}
		sessionID := createTestSession(t, ts.URL, "my/module", changes)

		resp, err := http.Get(ts.URL + "/api/reviews/" + sessionID)
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		var session ReviewSession
		if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if session.Status != "pending" {
			t.Errorf("status = %q, want pending", session.Status)
		}
		if session.Decision != "" {
			t.Errorf("decision = %q, want empty", session.Decision)
		}
		if len(session.ChangeComments) != 0 {
			t.Errorf("change_comments len = %d, want 0", len(session.ChangeComments))
		}
	})

	t.Run("not_found", func(t *testing.T) {
		resetReviewStore()
		root := t.TempDir()
		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/reviews/nonexistent-id")
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

func TestSubmitReview(t *testing.T) {
	t.Run("approve", func(t *testing.T) {
		resetReviewStore()
		root := t.TempDir()
		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		changes := []ChangeEntry{
			{ChangeID: "chg-001", FilePath: "changes/001_init.yaml", ChangeYAML: "id: chg-001"},
		}
		sessionID := createTestSession(t, ts.URL, "my/module", changes)

		submitBody, _ := json.Marshal(map[string]any{
			"decision":        "approved",
			"change_comments": map[string]string{},
		})
		resp, err := http.Post(ts.URL+"/api/reviews/"+sessionID+"/submit", "application/json", bytes.NewReader(submitBody))
		if err != nil {
			t.Fatalf("submit error: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("submit status = %d, want 200", resp.StatusCode)
		}

		// GET and verify
		resp, err = http.Get(ts.URL + "/api/reviews/" + sessionID)
		if err != nil {
			t.Fatalf("get error: %v", err)
		}
		defer resp.Body.Close()

		var session ReviewSession
		if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if session.Status != "submitted" {
			t.Errorf("status = %q, want submitted", session.Status)
		}
		if session.Decision != "approved" {
			t.Errorf("decision = %q, want approved", session.Decision)
		}
	})

	t.Run("reject_with_comments", func(t *testing.T) {
		resetReviewStore()
		root := t.TempDir()
		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		changes := []ChangeEntry{
			{ChangeID: "chg-001", FilePath: "changes/001_init.yaml", ChangeYAML: "id: chg-001"},
		}
		sessionID := createTestSession(t, ts.URL, "my/module", changes)

		submitBody, _ := json.Marshal(map[string]any{
			"decision": "rejected",
			"change_comments": map[string]string{
				"chg-001": "needs more detail on error handling",
			},
		})
		resp, err := http.Post(ts.URL+"/api/reviews/"+sessionID+"/submit", "application/json", bytes.NewReader(submitBody))
		if err != nil {
			t.Fatalf("submit error: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("submit status = %d, want 200", resp.StatusCode)
		}

		// GET and verify
		resp, err = http.Get(ts.URL + "/api/reviews/" + sessionID)
		if err != nil {
			t.Fatalf("get error: %v", err)
		}
		defer resp.Body.Close()

		var session ReviewSession
		if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if session.Status != "submitted" {
			t.Errorf("status = %q, want submitted", session.Status)
		}
		if session.Decision != "rejected" {
			t.Errorf("decision = %q, want rejected", session.Decision)
		}
		comment, ok := session.ChangeComments["chg-001"]
		if !ok {
			t.Fatal("change_comments missing key chg-001")
		}
		if comment != "needs more detail on error handling" {
			t.Errorf("comment = %q, want 'needs more detail on error handling'", comment)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		resetReviewStore()
		root := t.TempDir()
		ts := httptest.NewServer(newTestMux(root))
		defer ts.Close()

		submitBody, _ := json.Marshal(map[string]any{
			"decision":        "approved",
			"change_comments": map[string]string{},
		})
		resp, err := http.Post(ts.URL+"/api/reviews/nonexistent-id/submit", "application/json", bytes.NewReader(submitBody))
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 404 {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}
	})
}
