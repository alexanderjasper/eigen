package spec

import (
	"fmt"
	"sort"
	"strings"
)

// applyTextChange applies a TextChange to the current field value.
// fieldName is used in error messages.
func applyTextChange(fieldName, current string, tc TextChange) (string, error) {
	if !tc.IsSet() {
		return current, nil
	}
	if tc.IsFullReplace() {
		return tc.FullText(), nil
	}
	result := current
	for _, op := range tc.Ops() {
		switch op.Op {
		case "replace":
			i := strings.Index(result, op.Old)
			if i < 0 {
				return "", fmt.Errorf("replace: %q not found in %s field", op.Old, fieldName)
			}
			result = result[:i] + op.New + result[i+len(op.Old):]
		case "prepend":
			result = op.Text + result
		case "append":
			result = result + op.Text
		case "delete":
			i := strings.Index(result, op.Text)
			if i < 0 {
				return "", fmt.Errorf("delete: %q not found in %s field", op.Text, fieldName)
			}
			result = result[:i] + result[i+len(op.Text):]
		default:
			return "", fmt.Errorf("unknown op %q in %s field", op.Op, fieldName)
		}
	}
	return result, nil
}

// Project folds a slice of Changes into a SpecModule projection.
// path is the slash-separated path relative to the specs root (e.g. "spec-cli/cmd-new").
// Changes are applied in ascending sequence order.
func Project(path string, changes []*Change) (SpecModule, error) {
	sorted := make([]*Change, len(changes))
	copy(sorted, changes)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Sequence < sorted[j].Sequence
	})

	domain, module := pathSegments(path)
	s := SpecModule{
		ID:           path,
		Domain:       domain,
		Module:       module,
		Dependencies: []string{},
		Technology:   map[string]string{},
	}
	s.Format = "eigen/v1"

	// acMap tracks AC by id, preserving insertion order via acOrder.
	acMap := map[string]AcceptanceCriterion{}
	acOrder := []string{}

	allCompiled := len(sorted) > 0
	for _, ch := range sorted {
		if ch.Status != "compiled" {
			allCompiled = false
		}
		cs := ch.Changes

		if cs.Title != "" {
			s.Title = cs.Title
		}
		if cs.Owner != "" {
			s.Owner = cs.Owner
		}
		if cs.Status != "" {
			s.Status = cs.Status
			if cs.Status != "deprecated" {
				s.DeprecationReason = "" // clear when leaving deprecated
			}
		}
		if cs.DeprecationReason != "" {
			s.DeprecationReason = cs.DeprecationReason
		}
		if cs.Description.IsSet() {
			result, err := applyTextChange("description", s.Description, cs.Description)
			if err != nil {
				return SpecModule{}, fmt.Errorf("change %s: %w", ch.ID, err)
			}
			s.Description = result
		}
		if cs.Behavior.IsSet() {
			result, err := applyTextChange("behavior", s.Behavior, cs.Behavior)
			if err != nil {
				return SpecModule{}, fmt.Errorf("change %s: %w", ch.ID, err)
			}
			s.Behavior = result
		}
		if cs.Technology != nil {
			for k, v := range cs.Technology {
				s.Technology[k] = v
			}
		}
		if cs.Dependencies != nil {
			s.Dependencies = cs.Dependencies
		}

		for _, ac := range cs.AcceptanceCriteria {
			if _, exists := acMap[ac.ID]; !exists {
				acOrder = append(acOrder, ac.ID)
			}
			acMap[ac.ID] = ac
		}

		s.LastChange = ch.ID
		s.ChangesCount++
	}

	// If every change has been compiled and no change explicitly set a terminal
	// module status (deprecated/removed), promote the effective status to "compiled".
	if allCompiled && s.Status != "deprecated" && s.Status != "removed" {
		s.Status = "compiled"
	}

	// Build final AC list, excluding removed entries.
	var acs []AcceptanceCriterion
	for _, id := range acOrder {
		ac := acMap[id]
		if !ac.Removed {
			ac.Removed = false // don't write false to yaml
			acs = append(acs, ac)
		}
	}
	s.AcceptanceCriteria = acs

	return s, nil
}

// pathSegments returns the domain (first segment) and module (last segment) of a slash path.
// For a single-segment path like "projection-engine", both are equal.
func pathSegments(path string) (domain, module string) {
	parts := strings.Split(path, "/")
	return parts[0], parts[len(parts)-1]
}
