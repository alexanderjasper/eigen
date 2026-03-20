package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/alexanderjasper/eigen/internal/spec"
)

// ModuleRef identifies a module by its slash path relative to the specs root.
type ModuleRef struct {
	Path string // e.g. "spec-cli" or "spec-cli/cmd-new"
}

// ModulePath returns the absolute directory for a module.
func ModulePath(specsRoot, path string) string {
	return filepath.Join(specsRoot, filepath.FromSlash(path))
}

// EventsPath returns the events/ directory for a module.
func EventsPath(specsRoot, path string) string {
	return filepath.Join(ModulePath(specsRoot, path), "events")
}

// SpecPath returns the spec.yaml path for a module.
func SpecPath(specsRoot, path string) string {
	return filepath.Join(ModulePath(specsRoot, path), "spec.yaml")
}

// ReadEvents reads and parses all event YAML files from a module's events/ directory.
func ReadEvents(specsRoot, path string) ([]spec.ChangeEvent, error) {
	dir := EventsPath(specsRoot, path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading events dir %s: %w", dir, err)
	}

	var events []spec.ChangeEvent
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading event file %s: %w", e.Name(), err)
		}
		var ev spec.ChangeEvent
		if err := yaml.Unmarshal(data, &ev); err != nil {
			return nil, fmt.Errorf("parsing event file %s: %w", e.Name(), err)
		}
		events = append(events, ev)
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Sequence < events[j].Sequence
	})

	return events, nil
}

// WriteSpec marshals a SpecModule and writes it to spec.yaml.
func WriteSpec(specsRoot, path string, s spec.SpecModule) error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling spec: %w", err)
	}
	if err := os.WriteFile(SpecPath(specsRoot, path), data, 0644); err != nil {
		return fmt.Errorf("writing spec.yaml: %w", err)
	}
	return nil
}

// WriteEvent writes a ChangeEvent to the events/ directory with the given sequence number and slug.
func WriteEvent(specsRoot, path string, ev spec.ChangeEvent, slug string) error {
	dir := EventsPath(specsRoot, path)
	filename := fmt.Sprintf("%03d_%s.yaml", ev.Sequence, slug)
	data, err := yaml.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
		return fmt.Errorf("writing event file: %w", err)
	}
	return nil
}

// NextSequence returns the next sequence number for a module's events.
func NextSequence(specsRoot, path string) (int, error) {
	events, err := ReadEvents(specsRoot, path)
	if err != nil || len(events) == 0 {
		return 1, nil
	}
	max := 0
	for _, ev := range events {
		if ev.Sequence > max {
			max = ev.Sequence
		}
	}
	return max + 1, nil
}

// ReadSpec reads and parses the spec.yaml for a module.
func ReadSpec(specsRoot, path string) (spec.SpecModule, error) {
	data, err := os.ReadFile(SpecPath(specsRoot, path))
	if err != nil {
		return spec.SpecModule{}, fmt.Errorf("reading spec.yaml: %w", err)
	}
	var s spec.SpecModule
	if err := yaml.Unmarshal(data, &s); err != nil {
		return spec.SpecModule{}, fmt.Errorf("parsing spec.yaml: %w", err)
	}
	return s, nil
}

// WalkModules returns all ModuleRefs found under specsRoot at arbitrary depth.
// A directory is a module if it contains an events/ subdirectory.
// An optional prefix filters results to paths that start with the given prefix.
func WalkModules(specsRoot, prefix string) ([]ModuleRef, error) {
	var refs []ModuleRef
	err := walkDir(specsRoot, specsRoot, prefix, &refs)
	if err != nil {
		return nil, err
	}
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].Path < refs[j].Path
	})
	return refs, nil
}

func walkDir(specsRoot, dir, prefix string, refs *[]ModuleRef) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", dir, err)
	}

	hasEvents := false
	var subdirs []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if e.Name() == "events" {
			hasEvents = true
		} else {
			subdirs = append(subdirs, e)
		}
	}

	if hasEvents {
		rel, err := filepath.Rel(specsRoot, dir)
		if err != nil {
			return err
		}
		slashPath := filepath.ToSlash(rel)
		if prefix == "" || slashPath == prefix || strings.HasPrefix(slashPath, prefix+"/") {
			*refs = append(*refs, ModuleRef{Path: slashPath})
		}
	}

	for _, sub := range subdirs {
		if err := walkDir(specsRoot, filepath.Join(dir, sub.Name()), prefix, refs); err != nil {
			return err
		}
	}

	return nil
}
