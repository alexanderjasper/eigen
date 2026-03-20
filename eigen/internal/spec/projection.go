package spec

import (
	"sort"
	"strings"
)

// Project folds a slice of ChangeEvents into a SpecModule projection.
// path is the slash-separated path relative to the specs root (e.g. "spec-cli/cmd-new").
// Events are applied in ascending sequence order.
func Project(path string, events []ChangeEvent) SpecModule {
	sorted := make([]ChangeEvent, len(events))
	copy(sorted, events)
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

	// acMap tracks AC by id, preserving insertion order via acOrder.
	acMap := map[string]AcceptanceCriterion{}
	acOrder := []string{}

	for _, ev := range sorted {
		c := ev.Changes

		if c.Title != "" {
			s.Title = c.Title
		}
		if c.Owner != "" {
			s.Owner = c.Owner
		}
		if c.Status != "" {
			s.Status = c.Status
		}
		if c.Description != "" {
			s.Description = c.Description
		}
		if c.Behavior != "" {
			s.Behavior = c.Behavior
		}
		if c.Technology != nil {
			for k, v := range c.Technology {
				s.Technology[k] = v
			}
		}
		if c.Dependencies != nil {
			s.Dependencies = c.Dependencies
		}

		for _, ac := range c.AcceptanceCriteria {
			if _, exists := acMap[ac.ID]; !exists {
				acOrder = append(acOrder, ac.ID)
			}
			acMap[ac.ID] = ac
		}

		s.LastEvent = ev.ID
		s.EventsCount++
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
