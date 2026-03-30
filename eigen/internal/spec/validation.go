package spec

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LintError describes a YAML authoring error found in a change file.
type LintError struct {
	File    string
	Line    int
	Message string
}

func (e LintError) Error() string {
	return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
}

// LintChangeFile scans a change file's raw bytes for common YAML authoring errors:
//   - A backtick character inside an unquoted scalar value
//   - A bare ": " or trailing ":" inside an unquoted scalar value
//
// It skips lines that are part of a block scalar (introduced by | or >).
func LintChangeFile(filePath string, data []byte) []LintError {
	var errs []LintError

	lines := bytes.Split(data, []byte("\n"))
	blockScalarIndent := -1 // indent level of the block scalar header; -1 means not in block scalar

	for lineNum, rawLine := range lines {
		lineNo := lineNum + 1
		line := string(rawLine)

		// Determine current line's indentation.
		trimmed := strings.TrimLeft(line, " \t")
		indent := len(line) - len(trimmed)

		// If we're inside a block scalar, check whether this line exits it.
		if blockScalarIndent >= 0 {
			// A non-empty line at indent <= block scalar header indent exits the block scalar.
			// Empty / whitespace-only lines remain inside.
			if len(strings.TrimSpace(line)) > 0 && indent <= blockScalarIndent {
				blockScalarIndent = -1
			} else {
				continue // still inside block scalar — skip lint
			}
		}

		// Look for a YAML key: value pattern on this line.
		// We only care about the value portion of a mapping line.
		colonIdx := strings.Index(trimmed, ": ")
		var valuePart string
		if colonIdx >= 0 {
			valuePart = strings.TrimSpace(trimmed[colonIdx+2:])
		} else if strings.HasSuffix(strings.TrimSpace(trimmed), ":") {
			// Key with no inline value — no value to lint.
			continue
		} else {
			// Not a mapping line (list item or continuation) — skip.
			continue
		}

		// Detect block scalar start: value is "|" or ">" with optional chomping/indent indicators.
		stripped := strings.TrimRight(valuePart, " \t")
		if stripped == "|" || stripped == ">" ||
			strings.HasPrefix(stripped, "|+") || strings.HasPrefix(stripped, "|-") ||
			strings.HasPrefix(stripped, ">+") || strings.HasPrefix(stripped, ">-") {
			blockScalarIndent = indent
			continue
		}

		// Skip empty values and quoted / flow values — handled by the YAML parser.
		if len(valuePart) == 0 {
			continue
		}
		first := valuePart[0]
		if first == '"' || first == '\'' || first == '[' || first == '{' {
			continue
		}

		// valuePart is an unquoted scalar. Apply lint rules.
		if strings.ContainsRune(valuePart, '`') {
			errs = append(errs, LintError{
				File:    filePath,
				Line:    lineNo,
				Message: "backtick character in unquoted scalar (wrap the value in double quotes)",
			})
		}
		if strings.Contains(valuePart, ": ") || strings.HasSuffix(strings.TrimRight(valuePart, " \t"), ":") {
			errs = append(errs, LintError{
				File:    filePath,
				Line:    lineNo,
				Message: "bare colon in unquoted scalar would be parsed as a YAML key separator (wrap the value in double quotes)",
			})
		}
	}

	return errs
}

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

// ValidationWarning describes a non-fatal issue found during validation.
type ValidationWarning struct {
	Field   string
	Message string
}

func (w ValidationWarning) String() string {
	if w.Field != "" {
		return fmt.Sprintf("%s: %s", w.Field, w.Message)
	}
	return w.Message
}

// Validate checks a SpecModule for required fields, AC completeness, and dependency existence.
// specsRoot is the root specs/ directory used to resolve dependency module ids.
// It returns validation errors (blocking) and warnings (non-blocking).
func Validate(s SpecModule, specsRoot string) ([]ValidationError, []ValidationWarning) {
	var errs []ValidationError
	var warnings []ValidationWarning

	if s.Format == "" {
		warnings = append(warnings, ValidationWarning{Field: "format", Message: `format field is absent; expected "eigen/v1"`})
	}

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
			continue
		}
		// Check if the dependency's spec.yaml is deprecated.
		depSpecPath := filepath.Join(depPath, "spec.yaml")
		if depData, readErr := os.ReadFile(depSpecPath); readErr == nil {
			var depSpec struct {
				Status string `yaml:"status"`
			}
			if yaml.Unmarshal(depData, &depSpec) == nil && depSpec.Status == "deprecated" {
				warnings = append(warnings, ValidationWarning{
					Field:   "dependencies",
					Message: fmt.Sprintf("module %q is deprecated", dep),
				})
			}
		}
	}

	return errs, warnings
}

// ValidateChangeLog replays changes in order and returns a ValidationError for each
// field in a changeset that is identical to the spec state just before that change.
// path is passed through to Project for the initial state.
func ValidateChangeLog(path string, changes []*Change) []ValidationError {
	var errs []ValidationError
	current := SpecModule{Dependencies: []string{}, Technology: map[string]string{}}
	for _, ch := range changes {
		for _, e := range ValidateChanges(current, ch.Changes) {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("change %s: %s", ch.ID, e.Field),
				Message: e.Message,
			})
		}
		current = Project(path, changes[:indexOf(changes, *ch)+1])
	}
	return errs
}

func indexOf(changes []*Change, target Change) int {
	for i, ch := range changes {
		if ch.ID == target.ID && ch.Sequence == target.Sequence {
			return i
		}
	}
	return len(changes) - 1
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
