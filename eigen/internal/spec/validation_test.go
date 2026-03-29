package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validModule returns a fully-populated SpecModule that passes Validate.
func validModule() SpecModule {
	return SpecModule{
		ID:          "d/m",
		Domain:      "d",
		Module:      "m",
		Owner:       "alice",
		Title:       "My Module",
		Status:      "draft",
		Description: "A description",
		Behavior:    "Some behavior",
		AcceptanceCriteria: []AcceptanceCriterion{
			{ID: "AC-001", Description: "desc", Given: "g", When: "w", Then: "t"},
		},
		Dependencies: []string{},
		Technology:   map[string]string{},
	}
}

func TestValidate(t *testing.T) {
	t.Run("missing_required_fields", func(t *testing.T) {
		s := SpecModule{}
		errs := Validate(s, t.TempDir())

		// Expect errors for: id, domain, module, owner, title, description, behavior
		wantFields := map[string]bool{
			"id": false, "domain": false, "module": false,
			"owner": false, "title": false, "description": false, "behavior": false,
		}
		for _, e := range errs {
			if _, ok := wantFields[e.Field]; ok {
				wantFields[e.Field] = true
			}
		}
		for field, found := range wantFields {
			if !found {
				t.Errorf("expected error for field %q, but none found", field)
			}
		}
		if len(errs) < 7 {
			t.Errorf("got %d errors, want at least 7", len(errs))
		}
	})

	t.Run("missing_ac_subfields", func(t *testing.T) {
		s := validModule()
		s.AcceptanceCriteria = []AcceptanceCriterion{
			{ID: "AC-X", Description: "desc"}, // missing given, when, then
		}
		errs := Validate(s, t.TempDir())

		wantFields := map[string]bool{
			"acceptance_criteria[AC-X].given": false,
			"acceptance_criteria[AC-X].when":  false,
			"acceptance_criteria[AC-X].then":  false,
		}
		for _, e := range errs {
			if _, ok := wantFields[e.Field]; ok {
				wantFields[e.Field] = true
			}
		}
		for field, found := range wantFields {
			if !found {
				t.Errorf("expected error for field %q, but none found", field)
			}
		}
	})

	t.Run("broken_dependency", func(t *testing.T) {
		s := validModule()
		s.Dependencies = []string{"nonexistent/module"}
		specsRoot := t.TempDir()

		errs := Validate(s, specsRoot)

		found := false
		for _, e := range errs {
			if e.Field == "dependencies" && strings.Contains(e.Message, "nonexistent/module") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected dependency error mentioning path, got %v", errs)
		}
	})

	t.Run("valid_dependency", func(t *testing.T) {
		s := validModule()
		s.Dependencies = []string{"real/module"}
		specsRoot := t.TempDir()

		// Create the dependency directory
		depPath := filepath.Join(specsRoot, "real", "module")
		if err := os.MkdirAll(depPath, 0o755); err != nil {
			t.Fatal(err)
		}

		errs := Validate(s, specsRoot)

		for _, e := range errs {
			if e.Field == "dependencies" {
				t.Errorf("unexpected dependency error: %v", e)
			}
		}
	})
}

func TestValidateChanges(t *testing.T) {
	t.Run("noop_scalar_flagged", func(t *testing.T) {
		current := SpecModule{
			Title:        "X",
			Dependencies: []string{},
			Technology:   map[string]string{},
		}
		cs := ChangeSet{Title: "X"}

		errs := ValidateChanges(current, cs)

		found := false
		for _, e := range errs {
			if e.Field == "title" && strings.Contains(e.Message, "no-op") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected no-op error for title, got %v", errs)
		}
	})

	t.Run("identical_ac_flagged", func(t *testing.T) {
		current := SpecModule{
			AcceptanceCriteria: []AcceptanceCriterion{
				{ID: "AC-001", Description: "desc", Given: "g", When: "w", Then: "t"},
			},
			Dependencies: []string{},
			Technology:   map[string]string{},
		}
		cs := ChangeSet{
			AcceptanceCriteria: []AcceptanceCriterion{
				{ID: "AC-001", Description: "desc", Given: "g", When: "w", Then: "t"},
			},
		}

		errs := ValidateChanges(current, cs)

		found := false
		for _, e := range errs {
			if e.Field == "acceptance_criteria[AC-001]" && strings.Contains(e.Message, "no-op") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected no-op error for acceptance_criteria[AC-001], got %v", errs)
		}
	})
}

func TestLintChangeFile(t *testing.T) {
	t.Run("bare_colon_caught", func(t *testing.T) {
		data := []byte("summary: foo: bar\n")

		errs := LintChangeFile("test.yaml", data)

		found := false
		for _, e := range errs {
			if strings.Contains(e.Message, "bare colon") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected bare colon lint error, got %v", errs)
		}
	})

	t.Run("block_scalar_not_linted", func(t *testing.T) {
		data := []byte("description: |\n  this has: colons inside\n  and more: colons\n")

		errs := LintChangeFile("test.yaml", data)

		for _, e := range errs {
			if strings.Contains(e.Message, "bare colon") {
				t.Errorf("unexpected bare colon error inside block scalar: %v", e)
			}
		}
	})
}
