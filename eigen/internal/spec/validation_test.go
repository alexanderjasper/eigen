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
		errs, _ := Validate(s, t.TempDir())

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
		errs, _ := Validate(s, t.TempDir())

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

		errs, _ := Validate(s, specsRoot)

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

		errs, _ := Validate(s, specsRoot)

		for _, e := range errs {
			if e.Field == "dependencies" {
				t.Errorf("unexpected dependency error: %v", e)
			}
		}
	})

	t.Run("missing_format_produces_warning", func(t *testing.T) {
		s := validModule()
		// Format is not set — should produce a warning, not an error.
		errs, warnings := Validate(s, t.TempDir())

		for _, e := range errs {
			if e.Field == "format" {
				t.Errorf("unexpected error for format field: %v", e)
			}
		}
		found := false
		for _, w := range warnings {
			if w.Field == "format" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected warning for missing format field, got %v", warnings)
		}
	})

	t.Run("present_format_no_warning", func(t *testing.T) {
		s := validModule()
		s.Format = "eigen/v1"
		_, warnings := Validate(s, t.TempDir())

		for _, w := range warnings {
			if w.Field == "format" {
				t.Errorf("unexpected warning for format field: %v", w)
			}
		}
	})

	t.Run("deprecated_dependency_emits_warning", func(t *testing.T) {
		// AC-032
		s := validModule()
		s.Dependencies = []string{"dep/module"}
		specsRoot := t.TempDir()

		depDir := filepath.Join(specsRoot, "dep", "module")
		if err := os.MkdirAll(depDir, 0o755); err != nil {
			t.Fatal(err)
		}
		depSpec := []byte("status: deprecated\n")
		if err := os.WriteFile(filepath.Join(depDir, "spec.yaml"), depSpec, 0o644); err != nil {
			t.Fatal(err)
		}

		_, warnings := Validate(s, specsRoot)

		found := false
		for _, w := range warnings {
			if w.Field == "dependencies" && strings.Contains(w.Message, "dep/module") && strings.Contains(w.Message, "deprecated") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected deprecation warning for dep/module, got warnings %v", warnings)
		}
	})

	t.Run("non_deprecated_dependency_no_warning", func(t *testing.T) {
		// AC-033
		s := validModule()
		s.Dependencies = []string{"dep/module"}
		specsRoot := t.TempDir()

		depDir := filepath.Join(specsRoot, "dep", "module")
		if err := os.MkdirAll(depDir, 0o755); err != nil {
			t.Fatal(err)
		}
		depSpec := []byte("status: draft\n")
		if err := os.WriteFile(filepath.Join(depDir, "spec.yaml"), depSpec, 0o644); err != nil {
			t.Fatal(err)
		}

		_, warnings := Validate(s, specsRoot)

		for _, w := range warnings {
			if w.Field == "dependencies" {
				t.Errorf("unexpected dependency warning: %v", w)
			}
		}
	})

	t.Run("removed_dependency_no_deprecation_warning", func(t *testing.T) {
		// AC-034
		s := validModule()
		s.Dependencies = []string{"dep/module"}
		specsRoot := t.TempDir()

		depDir := filepath.Join(specsRoot, "dep", "module")
		if err := os.MkdirAll(depDir, 0o755); err != nil {
			t.Fatal(err)
		}
		depSpec := []byte("status: removed\n")
		if err := os.WriteFile(filepath.Join(depDir, "spec.yaml"), depSpec, 0o644); err != nil {
			t.Fatal(err)
		}

		_, warnings := Validate(s, specsRoot)

		for _, w := range warnings {
			if w.Field == "dependencies" && strings.Contains(w.Message, "deprecated") {
				t.Errorf("unexpected deprecation warning for removed module: %v", w)
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

	// AC-050: op-based change that produces identical text is flagged as no-op
	t.Run("op_noop_flagged", func(t *testing.T) {
		current := SpecModule{
			Behavior:     "hello",
			Dependencies: []string{},
			Technology:   map[string]string{},
		}
		cs := ChangeSet{
			Behavior: NewTextChangeOps([]TextOp{
				{Op: "prepend", Text: ""},
			}),
		}

		errs := ValidateChanges(current, cs)

		found := false
		for _, e := range errs {
			if e.Field == "behavior" && strings.Contains(e.Message, "no-op") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected no-op error for behavior, got %v", errs)
		}
	})

	// AC-051: op-based change that modifies the text is not flagged
	t.Run("op_change_not_flagged", func(t *testing.T) {
		current := SpecModule{
			Behavior:     "hello world",
			Dependencies: []string{},
			Technology:   map[string]string{},
		}
		cs := ChangeSet{
			Behavior: NewTextChangeOps([]TextOp{
				{Op: "replace", Old: "hello", New: "goodbye"},
			}),
		}

		errs := ValidateChanges(current, cs)

		for _, e := range errs {
			if e.Field == "behavior" {
				t.Errorf("unexpected no-op error for behavior: %v", e)
			}
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
