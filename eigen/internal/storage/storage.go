package storage

import (
	"bytes"
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

// ChangesPath returns the changes/ directory for a module.
func ChangesPath(specsRoot, path string) string {
	return filepath.Join(ModulePath(specsRoot, path), "changes")
}

// SpecPath returns the spec.yaml path for a module.
func SpecPath(specsRoot, path string) string {
	return filepath.Join(ModulePath(specsRoot, path), "spec.yaml")
}

// ReadChanges reads and parses all change YAML files from a module's changes/ directory.
func ReadChanges(specsRoot, path string) ([]spec.Change, error) {
	dir := ChangesPath(specsRoot, path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading changes dir %s: %w", dir, err)
	}

	var changes []spec.Change
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading change file %s: %w", e.Name(), err)
		}
		var ch spec.Change
		if err := yaml.Unmarshal(data, &ch); err != nil {
			return nil, fmt.Errorf("parsing change file %s: %w", e.Name(), err)
		}
		ch.Filename = e.Name()
		changes = append(changes, ch)
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Sequence < changes[j].Sequence
	})

	return changes, nil
}

// WriteSpec marshals a SpecModule and writes it to spec.yaml.
func WriteSpec(specsRoot, path string, s spec.SpecModule) error {
	data, err := marshalCanonical(s)
	if err != nil {
		return fmt.Errorf("marshaling spec: %w", err)
	}
	if err := os.WriteFile(SpecPath(specsRoot, path), data, 0644); err != nil {
		return fmt.Errorf("writing spec.yaml: %w", err)
	}
	return nil
}

// WriteChange writes a Change to the changes/ directory with the given sequence number and slug.
func WriteChange(specsRoot, path string, ch spec.Change, slug string) error {
	dir := ChangesPath(specsRoot, path)
	filename := fmt.Sprintf("%03d_%s.yaml", ch.Sequence, slug)
	data, err := marshalCanonical(ch)
	if err != nil {
		return fmt.Errorf("marshaling change: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
		return fmt.Errorf("writing change file: %w", err)
	}
	return nil
}

// NextSequence returns the next sequence number for a module's changes.
func NextSequence(specsRoot, path string) (int, error) {
	changes, err := ReadChanges(specsRoot, path)
	if err != nil || len(changes) == 0 {
		return 1, nil
	}
	max := 0
	for _, ch := range changes {
		if ch.Sequence > max {
			max = ch.Sequence
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

// SetChangeStatus reads a change file by filename, sets its Status field, and writes it back.
func SetChangeStatus(specsRoot, modulePath, filename, status string) error {
	dir := ChangesPath(specsRoot, modulePath)
	path := filepath.Join(dir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading change file %s: %w", filename, err)
	}
	var ch spec.Change
	if err := yaml.Unmarshal(data, &ch); err != nil {
		return fmt.Errorf("parsing change file %s: %w", filename, err)
	}
	ch.Status = status
	out, err := marshalCanonical(ch)
	if err != nil {
		return fmt.Errorf("marshaling change file %s: %w", filename, err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("writing change file %s: %w", filename, err)
	}
	return nil
}

// SetChangeComment reads a change file by filename, sets its ReviewComment field, and writes it back.
func SetChangeComment(specsRoot, modulePath, filename, comment string) error {
	dir := ChangesPath(specsRoot, modulePath)
	path := filepath.Join(dir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading change file %s: %w", filename, err)
	}
	var ch spec.Change
	if err := yaml.Unmarshal(data, &ch); err != nil {
		return fmt.Errorf("parsing change file %s: %w", filename, err)
	}
	ch.ReviewComment = comment
	out, err := marshalCanonical(ch)
	if err != nil {
		return fmt.Errorf("marshaling change file %s: %w", filename, err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("writing change file %s: %w", filename, err)
	}
	return nil
}

// FilterChangesByStatus returns changes whose effective status matches the given status.
// An absent (empty string) status is treated as "draft" (AC-006).
func FilterChangesByStatus(changes []spec.Change, status string) []spec.Change {
	var out []spec.Change
	for _, ch := range changes {
		effective := ch.Status
		if effective == "" {
			effective = "draft"
		}
		if effective == status {
			out = append(out, ch)
		}
	}
	return out
}

// WalkModules returns all ModuleRefs found under specsRoot at arbitrary depth.
// A directory is a module if it contains a changes/ subdirectory.
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

// marshalCanonical marshals v to YAML with 2-space indentation and omits zero-value scalar fields.
func marshalCanonical(v interface{}) ([]byte, error) {
	var node yaml.Node
	if err := node.Encode(v); err != nil {
		return nil, err
	}
	pruneZeroScalars(&node)
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&node); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// pruneZeroScalars removes mapping entries whose value is an empty string or "0".
func pruneZeroScalars(node *yaml.Node) {
	if node.Kind == yaml.DocumentNode {
		for _, child := range node.Content {
			pruneZeroScalars(child)
		}
		return
	}
	if node.Kind != yaml.MappingNode {
		return
	}
	var kept []*yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, val := node.Content[i], node.Content[i+1]
		if val.Kind == yaml.ScalarNode && (val.Value == "" || val.Value == "0") {
			continue
		}
		pruneZeroScalars(val)
		kept = append(kept, key, val)
	}
	node.Content = kept
}

func walkDir(specsRoot, dir, prefix string, refs *[]ModuleRef) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", dir, err)
	}

	hasChanges := false
	var subdirs []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if e.Name() == "changes" {
			hasChanges = true
		} else {
			subdirs = append(subdirs, e)
		}
	}

	if hasChanges {
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
