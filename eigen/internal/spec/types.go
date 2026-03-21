package spec

// SpecModule is the projection — the current state of a module derived from all its changes.
type SpecModule struct {
	ID          string            `yaml:"id"           json:"id"`
	Domain      string            `yaml:"domain"       json:"domain"`
	Module      string            `yaml:"module"       json:"module"`
	Owner       string            `yaml:"owner"        json:"owner"`
	Title       string            `yaml:"title"        json:"title"`
	Status      string            `yaml:"status"       json:"status"`
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
	ID        string    `yaml:"id"        json:"id"`
	Sequence  int       `yaml:"sequence"  json:"sequence"`
	Timestamp string    `yaml:"timestamp" json:"timestamp"`
	Author    string    `yaml:"author"    json:"author"`
	Type      string    `yaml:"type"      json:"type"` // created | updated | deprecated
	Summary   string    `yaml:"summary"   json:"summary"`
	Reason    string    `yaml:"reason"    json:"reason"`
	Changes   ChangeSet `yaml:"changes"   json:"changes"`
}

// ChangeSet holds the fields that may change in a single change.
// All fields are pointers / omitempty so absent fields are distinguishable from zero values.
type ChangeSet struct {
	Title       string            `yaml:"title,omitempty"       json:"title,omitempty"`
	Owner       string            `yaml:"owner,omitempty"       json:"owner,omitempty"`
	Status      string            `yaml:"status,omitempty"      json:"status,omitempty"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Behavior    string            `yaml:"behavior,omitempty"    json:"behavior,omitempty"`
	Technology  map[string]string `yaml:"technology,omitempty"  json:"technology,omitempty"`
	Dependencies []string         `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	AcceptanceCriteria []AcceptanceCriterion `yaml:"acceptance_criteria,omitempty" json:"acceptance_criteria,omitempty"`
}
