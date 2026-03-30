package spec

import (
	"sort"
	"strings"
)

// Project folds a slice of Changes into a SpecModule projection.
// path is the slash-separated path relative to the specs root (e.g. "spec-cli/cmd-new").
// Changes are applied in ascending sequence order.
func Project(path string, changes []*Change) SpecModule {
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

	for _, ch := range sorted {
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
		if cs.Description != "" {
			s.Description = cs.Description
		}
		if cs.Behavior != "" {
			s.Behavior = cs.Behavior
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

	return s
}

// pathSegments returns the domain (first segment) and module (last segment) of a slash path.
// For a single-segment path like "projection-engine", both are equal.
func pathSegments(path string) (domain, module string) {
	parts := strings.Split(path, "/")
	return parts[0], parts[len(parts)-1]
}
