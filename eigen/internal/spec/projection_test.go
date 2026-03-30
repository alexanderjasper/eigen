package spec

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestProject(t *testing.T) {
	t.Run("empty_changes", func(t *testing.T) {
		got, err := Project("d/m", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Format != "eigen/v1" {
			t.Errorf("Format = %q, want %q", got.Format, "eigen/v1")
		}
		if got.ID != "d/m" {
			t.Errorf("ID = %q, want %q", got.ID, "d/m")
		}
		if got.Domain != "d" {
			t.Errorf("Domain = %q, want %q", got.Domain, "d")
		}
		if got.Module != "m" {
			t.Errorf("Module = %q, want %q", got.Module, "m")
		}
		if got.Title != "" {
			t.Errorf("Title = %q, want empty", got.Title)
		}
		if got.Owner != "" {
			t.Errorf("Owner = %q, want empty", got.Owner)
		}
		if got.Status != "" {
			t.Errorf("Status = %q, want empty", got.Status)
		}
		if got.Description != "" {
			t.Errorf("Description = %q, want empty", got.Description)
		}
		if got.Behavior != "" {
			t.Errorf("Behavior = %q, want empty", got.Behavior)
		}
		if len(got.Dependencies) != 0 {
			t.Errorf("Dependencies = %v, want empty", got.Dependencies)
		}
		if got.Dependencies == nil {
			t.Error("Dependencies is nil, want non-nil empty slice")
		}
		if len(got.Technology) != 0 {
			t.Errorf("Technology = %v, want empty", got.Technology)
		}
		if got.Technology == nil {
			t.Error("Technology is nil, want non-nil empty map")
		}
		if got.LastChange != "" {
			t.Errorf("LastChange = %q, want empty", got.LastChange)
		}
		if got.ChangesCount != 0 {
			t.Errorf("ChangesCount = %d, want 0", got.ChangesCount)
		}
	})

	t.Run("single_change_sets_all_fields", func(t *testing.T) {
		changes := []*Change{
			{
				ID:       "chg-001",
				Sequence: 1,
				Changes: ChangeSet{
					Title:       "My Title",
					Owner:       "alice",
					Status:      "draft",
					Description: NewTextChangeScalar("A description"),
					Behavior:    NewTextChangeScalar("Some behavior"),
					Technology:  map[string]string{"lang": "go"},
					Dependencies: []string{"dep-a"},
					AcceptanceCriteria: []AcceptanceCriterion{
						{ID: "AC-001", Description: "desc", Given: "g", When: "w", Then: "t"},
					},
				},
			},
		}

		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Format != "eigen/v1" {
			t.Errorf("Format = %q, want %q", got.Format, "eigen/v1")
		}
		if got.Title != "My Title" {
			t.Errorf("Title = %q, want %q", got.Title, "My Title")
		}
		if got.Owner != "alice" {
			t.Errorf("Owner = %q, want %q", got.Owner, "alice")
		}
		if got.Status != "draft" {
			t.Errorf("Status = %q, want %q", got.Status, "draft")
		}
		if got.Description != "A description" {
			t.Errorf("Description = %q, want %q", got.Description, "A description")
		}
		if got.Behavior != "Some behavior" {
			t.Errorf("Behavior = %q, want %q", got.Behavior, "Some behavior")
		}
		if len(got.Technology) != 1 || got.Technology["lang"] != "go" {
			t.Errorf("Technology = %v, want {lang: go}", got.Technology)
		}
		if len(got.Dependencies) != 1 || got.Dependencies[0] != "dep-a" {
			t.Errorf("Dependencies = %v, want [dep-a]", got.Dependencies)
		}
		if len(got.AcceptanceCriteria) != 1 || got.AcceptanceCriteria[0].ID != "AC-001" {
			t.Errorf("AcceptanceCriteria = %v, want [AC-001]", got.AcceptanceCriteria)
		}
		if got.LastChange != "chg-001" {
			t.Errorf("LastChange = %q, want %q", got.LastChange, "chg-001")
		}
		if got.ChangesCount != 1 {
			t.Errorf("ChangesCount = %d, want 1", got.ChangesCount)
		}
	})

	t.Run("last_write_wins_scalars", func(t *testing.T) {
		changes := []*Change{
			{
				ID:       "chg-001",
				Sequence: 1,
				Changes: ChangeSet{
					Title:       "First",
					Owner:       "alice",
					Status:      "draft",
					Description: NewTextChangeScalar("desc1"),
					Behavior:    NewTextChangeScalar("beh1"),
				},
			},
			{
				ID:       "chg-002",
				Sequence: 2,
				Changes: ChangeSet{
					Title:       "Second",
					Owner:       "bob",
					Status:      "approved",
					Description: NewTextChangeScalar("desc2"),
					Behavior:    NewTextChangeScalar("beh2"),
				},
			},
		}

		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Format != "eigen/v1" {
			t.Errorf("Format = %q, want %q", got.Format, "eigen/v1")
		}
		if got.Title != "Second" {
			t.Errorf("Title = %q, want %q", got.Title, "Second")
		}
		if got.Owner != "bob" {
			t.Errorf("Owner = %q, want %q", got.Owner, "bob")
		}
		if got.Status != "approved" {
			t.Errorf("Status = %q, want %q", got.Status, "approved")
		}
		if got.Description != "desc2" {
			t.Errorf("Description = %q, want %q", got.Description, "desc2")
		}
		if got.Behavior != "beh2" {
			t.Errorf("Behavior = %q, want %q", got.Behavior, "beh2")
		}
	})

	t.Run("ac_merged_by_id", func(t *testing.T) {
		changes := []*Change{
			{
				ID:       "chg-001",
				Sequence: 1,
				Changes: ChangeSet{
					AcceptanceCriteria: []AcceptanceCriterion{
						{ID: "AC-A", Description: "original", Given: "g", When: "w", Then: "t1"},
					},
				},
			},
			{
				ID:       "chg-002",
				Sequence: 2,
				Changes: ChangeSet{
					AcceptanceCriteria: []AcceptanceCriterion{
						{ID: "AC-B", Description: "new", Given: "g2", When: "w2", Then: "t2"},
						{ID: "AC-A", Description: "updated", Given: "g", When: "w", Then: "t-updated"},
					},
				},
			},
		}

		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Format != "eigen/v1" {
			t.Errorf("Format = %q, want %q", got.Format, "eigen/v1")
		}
		if len(got.AcceptanceCriteria) != 2 {
			t.Fatalf("len(AC) = %d, want 2", len(got.AcceptanceCriteria))
		}
		// Insertion order: AC-A first, then AC-B
		if got.AcceptanceCriteria[0].ID != "AC-A" {
			t.Errorf("AC[0].ID = %q, want AC-A", got.AcceptanceCriteria[0].ID)
		}
		if got.AcceptanceCriteria[0].Then != "t-updated" {
			t.Errorf("AC[0].Then = %q, want t-updated", got.AcceptanceCriteria[0].Then)
		}
		if got.AcceptanceCriteria[1].ID != "AC-B" {
			t.Errorf("AC[1].ID = %q, want AC-B", got.AcceptanceCriteria[1].ID)
		}
	})

	t.Run("ac_removed_dropped", func(t *testing.T) {
		changes := []*Change{
			{
				ID:       "chg-001",
				Sequence: 1,
				Changes: ChangeSet{
					AcceptanceCriteria: []AcceptanceCriterion{
						{ID: "AC-A", Description: "desc", Given: "g", When: "w", Then: "t"},
					},
				},
			},
			{
				ID:       "chg-002",
				Sequence: 2,
				Changes: ChangeSet{
					AcceptanceCriteria: []AcceptanceCriterion{
						{ID: "AC-A", Removed: true},
					},
				},
			},
		}

		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Format != "eigen/v1" {
			t.Errorf("Format = %q, want %q", got.Format, "eigen/v1")
		}
		if len(got.AcceptanceCriteria) != 0 {
			t.Errorf("len(AC) = %d, want 0", len(got.AcceptanceCriteria))
		}
	})

	t.Run("dependencies_replaced", func(t *testing.T) {
		changes := []*Change{
			{
				ID:       "chg-001",
				Sequence: 1,
				Changes: ChangeSet{
					Dependencies: []string{"A", "B"},
				},
			},
			{
				ID:       "chg-002",
				Sequence: 2,
				Changes: ChangeSet{
					Dependencies: []string{"C"},
				},
			},
		}

		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Format != "eigen/v1" {
			t.Errorf("Format = %q, want %q", got.Format, "eigen/v1")
		}
		if len(got.Dependencies) != 1 || got.Dependencies[0] != "C" {
			t.Errorf("Dependencies = %v, want [C]", got.Dependencies)
		}
	})

	t.Run("technology_merged", func(t *testing.T) {
		changes := []*Change{
			{
				ID:       "chg-001",
				Sequence: 1,
				Changes: ChangeSet{
					Technology: map[string]string{"lang": "go"},
				},
			},
			{
				ID:       "chg-002",
				Sequence: 2,
				Changes: ChangeSet{
					Technology: map[string]string{"db": "postgres"},
				},
			},
		}

		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Format != "eigen/v1" {
			t.Errorf("Format = %q, want %q", got.Format, "eigen/v1")
		}
		if len(got.Technology) != 2 {
			t.Fatalf("len(Technology) = %d, want 2", len(got.Technology))
		}
		if got.Technology["lang"] != "go" {
			t.Errorf("Technology[lang] = %q, want go", got.Technology["lang"])
		}
		if got.Technology["db"] != "postgres" {
			t.Errorf("Technology[db] = %q, want postgres", got.Technology["db"])
		}
	})

	t.Run("metadata_reflects_final", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{Title: "t1"}},
			{ID: "chg-002", Sequence: 2, Changes: ChangeSet{Title: "t2"}},
			{ID: "chg-003", Sequence: 3, Changes: ChangeSet{Title: "t3"}},
		}

		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Format != "eigen/v1" {
			t.Errorf("Format = %q, want %q", got.Format, "eigen/v1")
		}
		if got.LastChange != "chg-003" {
			t.Errorf("LastChange = %q, want chg-003", got.LastChange)
		}
		if got.ChangesCount != 3 {
			t.Errorf("ChangesCount = %d, want 3", got.ChangesCount)
		}
	})

	t.Run("sequence_ordering", func(t *testing.T) {
		// Provide in disorder: seq 3, seq 1, seq 2
		changes := []*Change{
			{ID: "chg-003", Sequence: 3, Changes: ChangeSet{Title: "third"}},
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{Title: "first"}},
			{ID: "chg-002", Sequence: 2, Changes: ChangeSet{Title: "second"}},
		}

		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Format != "eigen/v1" {
			t.Errorf("Format = %q, want %q", got.Format, "eigen/v1")
		}
		// Applied in order 1,2,3 so last-write-wins means title="third"
		if got.Title != "third" {
			t.Errorf("Title = %q, want %q", got.Title, "third")
		}
		if got.LastChange != "chg-003" {
			t.Errorf("LastChange = %q, want chg-003", got.LastChange)
		}
		if got.ChangesCount != 3 {
			t.Errorf("ChangesCount = %d, want 3", got.ChangesCount)
		}
	})

	t.Run("format_not_sourced_from_changes", func(t *testing.T) {
		// Even if a Change carries a Format field, Project must always return "eigen/v1".
		// (ChangeSet has no Format field so the loop can never overwrite it — this test
		// documents the intent explicitly.)
		changes := []*Change{
			{
				ID:       "chg-001",
				Sequence: 1,
				Format:   "some-other-version",
				Changes:  ChangeSet{Title: "t1"},
			},
		}

		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Format != "eigen/v1" {
			t.Errorf("Format = %q, want %q (must not be sourced from change)", got.Format, "eigen/v1")
		}
	})

	t.Run("deprecated_status_projected", func(t *testing.T) {
		// AC-028
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{Status: "deprecated"}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != "deprecated" {
			t.Errorf("Status = %q, want %q", got.Status, "deprecated")
		}
	})

	t.Run("removed_status_projected", func(t *testing.T) {
		// AC-029
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{Status: "removed"}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != "removed" {
			t.Errorf("Status = %q, want %q", got.Status, "removed")
		}
	})

	t.Run("deprecation_reason_preserved", func(t *testing.T) {
		// AC-030
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{
				Status:            "deprecated",
				DeprecationReason: "Use module foo instead",
			}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.DeprecationReason != "Use module foo instead" {
			t.Errorf("DeprecationReason = %q, want %q", got.DeprecationReason, "Use module foo instead")
		}
	})

	t.Run("deprecation_reason_cleared_on_status_change", func(t *testing.T) {
		// AC-031
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{
				Status:            "deprecated",
				DeprecationReason: "Use module foo instead",
			}},
			{ID: "chg-002", Sequence: 2, Changes: ChangeSet{
				Status: "draft",
			}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.DeprecationReason != "" {
			t.Errorf("DeprecationReason = %q, want empty after status change from deprecated", got.DeprecationReason)
		}
	})

	// AC-041: all changes compiled promotes module status
	t.Run("all_changes_compiled_promotes_status", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Status: "compiled", Changes: ChangeSet{Status: "draft"}},
			{ID: "chg-002", Sequence: 2, Status: "compiled", Changes: ChangeSet{Title: "updated"}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != "compiled" {
			t.Errorf("Status = %q, want %q (all changes compiled)", got.Status, "compiled")
		}
	})

	// AC-042: mixed compiled/non-compiled does not promote
	t.Run("mixed_compiled_status_no_promotion", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Status: "compiled", Changes: ChangeSet{Status: "draft"}},
			{ID: "chg-002", Sequence: 2, Status: "draft", Changes: ChangeSet{Title: "updated"}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != "draft" {
			t.Errorf("Status = %q, want %q (not all changes compiled)", got.Status, "draft")
		}
	})

	// AC-043: deprecated takes precedence over compiled promotion
	t.Run("all_compiled_deprecated_not_promoted", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Status: "compiled", Changes: ChangeSet{Status: "deprecated", DeprecationReason: "obsolete"}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != "deprecated" {
			t.Errorf("Status = %q, want deprecated (terminal status overrides compiled promotion)", got.Status)
		}
	})

	// AC-044: removed takes precedence over compiled promotion
	t.Run("all_compiled_removed_not_promoted", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Status: "compiled", Changes: ChangeSet{Status: "removed"}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != "removed" {
			t.Errorf("Status = %q, want removed (terminal status overrides compiled promotion)", got.Status)
		}
	})

	// AC-045: replace op substitutes first occurrence
	t.Run("replace_first_occurrence", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{
				Behavior: NewTextChangeScalar("foo bar foo"),
			}},
			{ID: "chg-002", Sequence: 2, Changes: ChangeSet{
				Behavior: NewTextChangeOps([]TextOp{
					{Op: "replace", Old: "foo", New: "baz"},
				}),
			}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Behavior != "baz bar foo" {
			t.Errorf("Behavior = %q, want %q", got.Behavior, "baz bar foo")
		}
	})

	// AC-046: prepend op inserts text before field value
	t.Run("prepend", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{
				Description: NewTextChangeScalar("existing text"),
			}},
			{ID: "chg-002", Sequence: 2, Changes: ChangeSet{
				Description: NewTextChangeOps([]TextOp{
					{Op: "prepend", Text: "new intro\n\n"},
				}),
			}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Description != "new intro\n\nexisting text" {
			t.Errorf("Description = %q, want %q", got.Description, "new intro\n\nexisting text")
		}
	})

	// AC-047: append op inserts text after field value
	t.Run("append", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{
				Description: NewTextChangeScalar("existing text"),
			}},
			{ID: "chg-002", Sequence: 2, Changes: ChangeSet{
				Description: NewTextChangeOps([]TextOp{
					{Op: "append", Text: "\n\nnew footer"},
				}),
			}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Description != "existing text\n\nnew footer" {
			t.Errorf("Description = %q, want %q", got.Description, "existing text\n\nnew footer")
		}
	})

	// AC-048: delete op removes first occurrence
	t.Run("delete_first_occurrence", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{
				Behavior: NewTextChangeScalar("A B C B"),
			}},
			{ID: "chg-002", Sequence: 2, Changes: ChangeSet{
				Behavior: NewTextChangeOps([]TextOp{
					{Op: "delete", Text: "B "},
				}),
			}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Behavior != "A C B" {
			t.Errorf("Behavior = %q, want %q", got.Behavior, "A C B")
		}
	})

	// AC-049: multiple ops in a single change applied sequentially
	t.Run("multi_op_sequential", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{
				Behavior: NewTextChangeScalar("A B C"),
			}},
			{ID: "chg-002", Sequence: 2, Changes: ChangeSet{
				Behavior: NewTextChangeOps([]TextOp{
					{Op: "replace", Old: "A", New: "X"},
					{Op: "append", Text: " D"},
				}),
			}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Behavior != "X B C D" {
			t.Errorf("Behavior = %q, want %q", got.Behavior, "X B C D")
		}
	})

	// AC-050: scalar string value is full replacement (backward compatible)
	t.Run("scalar_full_replace_backward_compat", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{
				Behavior: NewTextChangeScalar("old text"),
			}},
			{ID: "chg-002", Sequence: 2, Changes: ChangeSet{
				Behavior: NewTextChangeScalar("new text"),
			}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Behavior != "new text" {
			t.Errorf("Behavior = %q, want %q", got.Behavior, "new text")
		}
	})

	// AC-051: replace op with no match returns a projection error
	t.Run("replace_no_match_error", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{
				Behavior: NewTextChangeScalar("hello world"),
			}},
			{ID: "chg-002", Sequence: 2, Changes: ChangeSet{
				Behavior: NewTextChangeOps([]TextOp{
					{Op: "replace", Old: "xyz", New: "abc"},
				}),
			}},
		}
		_, err := Project("d/m", changes)
		if err == nil {
			t.Fatal("expected error for replace with no match, got nil")
		}
		if !strings.Contains(err.Error(), "xyz") {
			t.Errorf("error %q does not mention target %q", err.Error(), "xyz")
		}
	})

	// AC-052: delete op with no match returns a projection error
	t.Run("delete_no_match_error", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{
				Behavior: NewTextChangeScalar("hello world"),
			}},
			{ID: "chg-002", Sequence: 2, Changes: ChangeSet{
				Behavior: NewTextChangeOps([]TextOp{
					{Op: "delete", Text: "xyz"},
				}),
			}},
		}
		_, err := Project("d/m", changes)
		if err == nil {
			t.Fatal("expected error for delete with no match, got nil")
		}
		if !strings.Contains(err.Error(), "xyz") {
			t.Errorf("error %q does not mention target %q", err.Error(), "xyz")
		}
	})

	// AC-053: ops apply against current projected value, not original
	t.Run("ops_across_changes", func(t *testing.T) {
		changes := []*Change{
			{ID: "chg-001", Sequence: 1, Changes: ChangeSet{
				Description: NewTextChangeScalar("A B C"),
			}},
			{ID: "chg-002", Sequence: 2, Changes: ChangeSet{
				Description: NewTextChangeOps([]TextOp{
					{Op: "replace", Old: "B", New: "X"},
				}),
			}},
			{ID: "chg-003", Sequence: 3, Changes: ChangeSet{
				Description: NewTextChangeOps([]TextOp{
					{Op: "replace", Old: "X", New: "Y"},
				}),
			}},
		}
		got, err := Project("d/m", changes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Description != "A Y C" {
			t.Errorf("Description = %q, want %q", got.Description, "A Y C")
		}
	})
}

// TestProjectCompiledCommitsNotProjected verifies AC-009: compiled_commits on a Change is not
// folded into the projected SpecModule YAML.
func TestProjectCompiledCommitsNotProjected(t *testing.T) {
	changes := []*Change{
		{
			ID:              "chg-001",
			Sequence:        1,
			CompiledCommits: []string{"abc1234567890"},
			Changes: ChangeSet{
				Title: "My Title",
			},
		},
	}

	got, err := Project("d/m", changes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Marshal the result to YAML and check that compiled_commits is absent.
	data, err := yaml.Marshal(got)
	if err != nil {
		t.Fatalf("yaml.Marshal error: %v", err)
	}
	if strings.Contains(string(data), "compiled_commits") {
		t.Errorf("projected spec.yaml contains compiled_commits, want it absent:\n%s", string(data))
	}
}
