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

// ValidateEventLog replays events in order and returns a ValidationError for each
// field in a changeset that is identical to the spec state just before that event.
// path is passed through to Project for the initial state.
func ValidateEventLog(path string, events []ChangeEvent) []ValidationError {
	var errs []ValidationError
	current := SpecModule{Dependencies: []string{}, Technology: map[string]string{}}
	for _, ev := range events {
		for _, e := range ValidateChanges(current, ev.Changes) {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("event %s: %s", ev.ID, e.Field),
				Message: e.Message,
			})
		}
		current = Project(path, events[:indexOf(events, ev)+1])
	}
	return errs
}

func indexOf(events []ChangeEvent, target ChangeEvent) int {
	for i, ev := range events {
		if ev.ID == target.ID && ev.Sequence == target.Sequence {
			return i
		}
	}
	return len(events) - 1
}

// ValidateChanges checks a ChangeSet against the current SpecModule projection and
// returns a ValidationError for each field whose value is identical to the current state.
func ValidateChanges(current SpecModule, changes ChangeSet) []ValidationError {
	var errs []ValidationError

	simpleFields := []struct {
		name    string
		changed string
		current string
	}{
		{"title", changes.Title, current.Title},
		{"owner", changes.Owner, current.Owner},
		{"status", changes.Status, current.Status},
		{"description", changes.Description, current.Description},
		{"behavior", changes.Behavior, current.Behavior},
	}
	for _, f := range simpleFields {
		if f.changed != "" && f.changed == f.current {
			errs = append(errs, ValidationError{
				Field:   f.name,
				Message: "no-op: value is identical to current spec state",
			})
		}
	}

	// Build a map of current ACs by id for O(1) lookup.
	currentACs := make(map[string]AcceptanceCriterion, len(current.AcceptanceCriteria))
	for _, ac := range current.AcceptanceCriteria {
		currentACs[ac.ID] = ac
	}

	for _, ac := range changes.AcceptanceCriteria {
		existing, exists := currentACs[ac.ID]
		if !exists {
			continue // new AC — always allowed
		}
		if ac.Description == existing.Description &&
			ac.Given == existing.Given &&
			ac.When == existing.When &&
			ac.Then == existing.Then {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("acceptance_criteria[%s]", ac.ID),
				Message: "no-op: criterion is identical to current spec state",
			})
		}
	}

	return errs
}
