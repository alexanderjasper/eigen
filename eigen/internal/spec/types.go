package spec

// SpecModule is the projection — the current state of a module derived from all its events.
type SpecModule struct {
	ID          string            `yaml:"id"`
	Domain      string            `yaml:"domain"`
	Module      string            `yaml:"module"`
	Owner       string            `yaml:"owner"`
	Title       string            `yaml:"title"`
	Status      string            `yaml:"status"`
	Description string            `yaml:"description"`
	Behavior    string            `yaml:"behavior"`
	AcceptanceCriteria []AcceptanceCriterion `yaml:"acceptance_criteria,omitempty"`
	Dependencies []string         `yaml:"dependencies"`
	Technology  map[string]string `yaml:"technology"`
	// metadata
	LastEvent   string `yaml:"last_event"`
	EventsCount int    `yaml:"events_count"`
}

// AcceptanceCriterion is a single verifiable behavior assertion.
type AcceptanceCriterion struct {
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
	Given       string `yaml:"given"`
	When        string `yaml:"when"`
	Then        string `yaml:"then"`
	Removed     bool   `yaml:"removed,omitempty"`
}

// ChangeEvent represents a single immutable change recorded against a module.
type ChangeEvent struct {
	ID       string `yaml:"id"`
	Sequence int    `yaml:"sequence"`
	Timestamp string `yaml:"timestamp"`
	Author   string `yaml:"author"`
	Type     string `yaml:"type"` // created | updated | deprecated
	Summary  string `yaml:"summary"`
	Reason   string `yaml:"reason"`
	Changes  ChangeSet `yaml:"changes"`
}

// ChangeSet holds the fields that may change in a single event.
// All fields are pointers / omitempty so absent fields are distinguishable from zero values.
type ChangeSet struct {
	Title       string            `yaml:"title,omitempty"`
	Owner       string            `yaml:"owner,omitempty"`
	Status      string            `yaml:"status,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Behavior    string            `yaml:"behavior,omitempty"`
	Technology  map[string]string `yaml:"technology,omitempty"`
	Dependencies []string         `yaml:"dependencies,omitempty"`
	AcceptanceCriteria []AcceptanceCriterion `yaml:"acceptance_criteria,omitempty"`
}
