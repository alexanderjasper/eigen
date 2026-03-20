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

// ModuleRef identifies a module by domain and module name.
type ModuleRef struct {
	Domain string
	Module string
}

// ModulePath returns the directory for a module under specsRoot.
func ModulePath(specsRoot, domain, module string) string {
	return filepath.Join(specsRoot, domain, module)
}

// EventsPath returns the events/ directory for a module.
func EventsPath(specsRoot, domain, module string) string {
	return filepath.Join(ModulePath(specsRoot, domain, module), "events")
}

// SpecPath returns the spec.yaml path for a module.
func SpecPath(specsRoot, domain, module string) string {
	return filepath.Join(ModulePath(specsRoot, domain, module), "spec.yaml")
}

// ReadEvents reads and parses all event YAML files from a module's events/ directory.
func ReadEvents(specsRoot, domain, module string) ([]spec.ChangeEvent, error) {
	dir := EventsPath(specsRoot, domain, module)
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
func WriteSpec(specsRoot, domain, module string, s spec.SpecModule) error {
	path := SpecPath(specsRoot, domain, module)
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling spec: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing spec.yaml: %w", err)
	}
	return nil
}

// WriteEvent writes a ChangeEvent to the events/ directory with the next sequence number.
func WriteEvent(specsRoot, domain, module string, ev spec.ChangeEvent, slug string) error {
	dir := EventsPath(specsRoot, domain, module)
	filename := fmt.Sprintf("%03d_%s.yaml", ev.Sequence, slug)
	path := filepath.Join(dir, filename)
	data, err := yaml.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing event file: %w", err)
	}
	return nil
}

// NextSequence returns the next sequence number for a module's events.
func NextSequence(specsRoot, domain, module string) (int, error) {
	events, err := ReadEvents(specsRoot, domain, module)
	if err != nil {
		// If the events dir doesn't exist yet, start at 1.
		return 1, nil
	}
	if len(events) == 0 {
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
func ReadSpec(specsRoot, domain, module string) (spec.SpecModule, error) {
	path := SpecPath(specsRoot, domain, module)
	data, err := os.ReadFile(path)
	if err != nil {
		return spec.SpecModule{}, fmt.Errorf("reading spec.yaml: %w", err)
	}
	var s spec.SpecModule
	if err := yaml.Unmarshal(data, &s); err != nil {
		return spec.SpecModule{}, fmt.Errorf("parsing spec.yaml: %w", err)
	}
	return s, nil
}

// WalkModules returns all ModuleRefs found under specsRoot.
// Optional domain filter: if non-empty, only modules in that domain are returned.
func WalkModules(specsRoot, domainFilter string) ([]ModuleRef, error) {
	var refs []ModuleRef

	domains, err := os.ReadDir(specsRoot)
	if err != nil {
		return nil, fmt.Errorf("reading specs root %s: %w", specsRoot, err)
	}

	for _, d := range domains {
		if !d.IsDir() {
			continue
		}
		domain := d.Name()
		if domainFilter != "" && domain != domainFilter {
			continue
		}
		modules, err := os.ReadDir(filepath.Join(specsRoot, domain))
		if err != nil {
			continue
		}
		for _, m := range modules {
			if !m.IsDir() {
				continue
			}
			refs = append(refs, ModuleRef{Domain: domain, Module: m.Name()})
		}
	}

	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Domain != refs[j].Domain {
			return refs[i].Domain < refs[j].Domain
		}
		return refs[i].Module < refs[j].Module
	})

	return refs, nil
}
