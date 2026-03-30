package spec

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// TextOp is a single operation applied to a text field.
type TextOp struct {
	Op   string `yaml:"op"             json:"op"`
	Old  string `yaml:"old,omitempty"  json:"old,omitempty"`
	New  string `yaml:"new,omitempty"  json:"new,omitempty"`
	Text string `yaml:"text,omitempty" json:"text,omitempty"`
}

// TextChange holds either a full replacement string or a sequence of TextOps.
// It is used for the Description and Behavior fields of ChangeSet.
// TextChange implements yaml.Unmarshaler and json.Unmarshaler to accept either
// a scalar (full replacement) or a sequence (ordered ops).
type TextChange struct {
	set      bool
	fullText string
	ops      []TextOp
}

// NewTextChangeScalar returns a TextChange representing a full replacement.
func NewTextChangeScalar(s string) TextChange {
	return TextChange{set: true, fullText: s}
}

// NewTextChangeOps returns a TextChange representing an ordered list of ops.
func NewTextChangeOps(ops []TextOp) TextChange {
	return TextChange{set: true, ops: ops}
}

// IsZero returns true when the TextChange was not set (for yaml omitempty support).
func (tc TextChange) IsZero() bool {
	return !tc.set
}

// IsSet returns true if the YAML key was present.
func (tc TextChange) IsSet() bool {
	return tc.set
}

// IsFullReplace returns true when the value is a scalar string.
func (tc TextChange) IsFullReplace() bool {
	return tc.set && tc.ops == nil
}

// FullText returns the scalar value.
func (tc TextChange) FullText() string {
	return tc.fullText
}

// Ops returns the ops list.
func (tc TextChange) Ops() []TextOp {
	return tc.ops
}

// UnmarshalYAML implements yaml.Unmarshaler.
// A scalar node is interpreted as a full replacement; a sequence node is interpreted as ops.
func (tc *TextChange) UnmarshalYAML(node *yaml.Node) error {
	tc.set = true
	switch node.Kind {
	case yaml.ScalarNode:
		tc.fullText = node.Value
		tc.ops = nil
		return nil
	case yaml.SequenceNode:
		var ops []TextOp
		if err := node.Decode(&ops); err != nil {
			return err
		}
		tc.ops = ops
		tc.fullText = ""
		return nil
	default:
		return fmt.Errorf("TextChange: expected scalar or sequence, got %v", node.Kind)
	}
}

// MarshalYAML implements yaml.Marshaler.
// Unset returns nil; scalar returns string; ops returns slice.
func (tc TextChange) MarshalYAML() (interface{}, error) {
	if !tc.set {
		return nil, nil
	}
	if tc.ops != nil {
		return tc.ops, nil
	}
	return tc.fullText, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (tc *TextChange) UnmarshalJSON(data []byte) error {
	// null → unset
	if string(data) == "null" {
		return nil
	}
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		tc.set = true
		tc.fullText = s
		tc.ops = nil
		return nil
	}
	// Try slice of TextOp
	var ops []TextOp
	if err := json.Unmarshal(data, &ops); err == nil {
		tc.set = true
		tc.ops = ops
		tc.fullText = ""
		return nil
	}
	return fmt.Errorf("TextChange: expected string or array, got %s", string(data))
}

// MarshalJSON implements json.Marshaler.
func (tc TextChange) MarshalJSON() ([]byte, error) {
	if !tc.set {
		return []byte("null"), nil
	}
	if tc.ops != nil {
		return json.Marshal(tc.ops)
	}
	return json.Marshal(tc.fullText)
}

// SpecModule is the projection — the current state of a module derived from all its changes.
type SpecModule struct {
	Format      string            `yaml:"format,omitempty" json:"format,omitempty"`
	ID          string            `yaml:"id"           json:"id"`
	Domain      string            `yaml:"domain"       json:"domain"`
	Module      string            `yaml:"module"       json:"module"`
	Owner       string            `yaml:"owner"        json:"owner"`
	Title       string            `yaml:"title"        json:"title"`
	Status            string `yaml:"status"       json:"status"`
	DeprecationReason string `yaml:"deprecation_reason,omitempty" json:"deprecation_reason,omitempty"`
	Description string            `yaml:"description"  json:"description"`
	Behavior    string            `yaml:"behavior"     json:"behavior"`
	AcceptanceCriteria []AcceptanceCriterion `yaml:"acceptance_criteria,omitempty" json:"acceptance_criteria,omitempty"`
	Dependencies []string         `yaml:"dependencies" json:"dependencies"`
	Technology  map[string]string `yaml:"technology"   json:"technology"`
	// metadata
	LastChange  string `yaml:"last_change"  json:"last_change"`
	ChangesCount int    `yaml:"changes_count" json:"changes_count"`
}

// AcceptanceCriterion is a single verifiable behavior assertion.
type AcceptanceCriterion struct {
	ID          string `yaml:"id"          json:"id"`
	Description string `yaml:"description" json:"description"`
	Given       string `yaml:"given"       json:"given"`
	When        string `yaml:"when"        json:"when"`
	Then        string `yaml:"then"        json:"then"`
	Removed     bool   `yaml:"removed,omitempty" json:"removed,omitempty"`
}

// Change represents a single immutable change recorded against a module.
type Change struct {
	Format    string    `yaml:"format,omitempty" json:"format,omitempty"`
	ID        string    `yaml:"id"        json:"id"`
	Sequence  int       `yaml:"sequence"  json:"sequence"`
	Timestamp string    `yaml:"timestamp" json:"timestamp"`
	Author    string    `yaml:"author"    json:"author"`
	Type      string    `yaml:"type"      json:"type"` // created | updated | deprecated
	Summary   string    `yaml:"summary"   json:"summary"`
	Reason    string    `yaml:"reason"    json:"reason"`
	Status          string   `yaml:"status,omitempty" json:"status,omitempty"`
	ReviewComment   string   `yaml:"review_comment,omitempty" json:"review_comment,omitempty"`
	CompiledCommits []string `yaml:"compiled_commits,omitempty" json:"compiled_commits,omitempty"`
	Filename        string   `yaml:"-" json:"filename,omitempty"`
	Changes       ChangeSet `yaml:"changes"   json:"changes"`
}

// ChangeSet holds the fields that may change in a single change.
// All fields are pointers / omitempty so absent fields are distinguishable from zero values.
type ChangeSet struct {
	Title       string            `yaml:"title,omitempty"       json:"title,omitempty"`
	Owner       string            `yaml:"owner,omitempty"       json:"owner,omitempty"`
	Status            string `yaml:"status,omitempty"            json:"status,omitempty"`
	DeprecationReason string `yaml:"deprecation_reason,omitempty" json:"deprecation_reason,omitempty"`
	Description TextChange        `yaml:"description,omitempty" json:"description,omitempty"`
	Behavior    TextChange        `yaml:"behavior,omitempty"    json:"behavior,omitempty"`
	Technology  map[string]string `yaml:"technology,omitempty"  json:"technology,omitempty"`
	Dependencies []string         `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	AcceptanceCriteria []AcceptanceCriterion `yaml:"acceptance_criteria,omitempty" json:"acceptance_criteria,omitempty"`
}
