package spec

import (
	"testing"
)

func TestProject(t *testing.T) {
	t.Run("empty_changes", func(t *testing.T) {
		got := Project("d/m", nil)

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
					Description: "A description",
					Behavior:    "Some behavior",
					Technology:  map[string]string{"lang": "go"},
					Dependencies: []string{"dep-a"},
					AcceptanceCriteria: []AcceptanceCriterion{
						{ID: "AC-001", Description: "desc", Given: "g", When: "w", Then: "t"},
					},
				},
			},
		}

		got := Project("d/m", changes)

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
					Description: "desc1",
					Behavior:    "beh1",
				},
			},
			{
				ID:       "chg-002",
				Sequence: 2,
				Changes: ChangeSet{
					Title:       "Second",
					Owner:       "bob",
					Status:      "approved",
					Description: "desc2",
					Behavior:    "beh2",
				},
			},
		}

		got := Project("d/m", changes)

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

		got := Project("d/m", changes)

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

		got := Project("d/m", changes)

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

		got := Project("d/m", changes)

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

		got := Project("d/m", changes)

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

		got := Project("d/m", changes)

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

		got := Project("d/m", changes)

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

		got := Project("d/m", changes)

		if got.Format != "eigen/v1" {
			t.Errorf("Format = %q, want %q (must not be sourced from change)", got.Format, "eigen/v1")
		}
	})
}
