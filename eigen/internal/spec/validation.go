package spec

import (
	"fmt"
	"os"
	"path/filepath"
)

// ValidationError describes a single validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// Validate checks a SpecModule for required fields, AC completeness, and dependency existence.
// specsRoot is the root specs/ directory used to resolve dependency module ids.
func Validate(s SpecModule, specsRoot string) []ValidationError {
	var errs []ValidationError

	required := []struct{ field, value string }{
		{"id", s.ID},
		{"domain", s.Domain},
		{"module", s.Module},
		{"owner", s.Owner},
		{"title", s.Title},
		{"description", s.Description},
		{"behavior", s.Behavior},
	}
	for _, r := range required {
		if r.value == "" {
			errs = append(errs, ValidationError{Field: r.field, Message: "required field is missing or empty"})
		}
	}

	for _, ac := range s.AcceptanceCriteria {
		acRequired := []struct{ field, value string }{
			{"id", ac.ID},
			{"description", ac.Description},
			{"given", ac.Given},
			{"when", ac.When},
			{"then", ac.Then},
		}
		for _, r := range acRequired {
			if r.value == "" {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("acceptance_criteria[%s].%s", ac.ID, r.field),
					Message: "required field is missing or empty",
				})
			}
		}
	}

	for _, dep := range s.Dependencies {
		// dep is a slash path, e.g. "spec-cli/cmd-new"
		depPath := filepath.Join(specsRoot, filepath.FromSlash(dep))
		if _, err := os.Stat(depPath); os.IsNotExist(err) {
			errs = append(errs, ValidationError{
				Field:   "dependencies",
				Message: fmt.Sprintf("module %q not found at %s", dep, depPath),
			})
		}
	}

	return errs
}
