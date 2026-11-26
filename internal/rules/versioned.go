package rules

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/agents"
)

// VersionedRule is an immutable rule version.
type VersionedRule struct {
	ID          string
	Version     int
	Description string
	Condition   string
	Action      string
	Source      string // "human", "feedback", "agent", "domain-pack"
	Active      bool
	CreatedAt   time.Time
	CreatedBy   string
	ParentID    string // previous version
}

// Engine is a versioned rule store.
// Rules are immutable once written. New versions supersede old ones.
type Engine struct {
	mu     sync.RWMutex
	rules  map[string][]VersionedRule // id -> versions (newest last)
	active map[string]int             // id -> active version number
}

func NewEngine() *Engine {
	return &Engine{
		rules:  make(map[string][]VersionedRule),
		active: make(map[string]int),
	}
}

// Propose creates a new version of a rule (or a new rule if ID is empty).
func (e *Engine) Propose(r VersionedRule) VersionedRule {
	e.mu.Lock()
	defer e.mu.Unlock()

	if r.ID == "" {
		r.ID = uuid.NewString()
		r.Version = 1
	} else {
		versions := e.rules[r.ID]
		r.Version = len(versions) + 1
		if len(versions) > 0 {
			r.ParentID = versions[len(versions)-1].ID + ":" + fmt.Sprint(versions[len(versions)-1].Version)
		}
	}
	r.CreatedAt = time.Now()
	r.Active = true

	e.rules[r.ID] = append(e.rules[r.ID], r)
	e.active[r.ID] = r.Version
	return r
}

// Activate sets which version of a rule is live.
func (e *Engine) Activate(id string, version int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	versions, ok := e.rules[id]
	if !ok {
		return fmt.Errorf("rule not found: %s", id)
	}
	found := false
	for i := range versions {
		if versions[i].Version == version {
			versions[i].Active = true
			found = true
		} else {
			versions[i].Active = false
		}
	}
	if !found {
		return fmt.Errorf("version %d not found for rule %s", version, id)
	}
	e.active[id] = version
	return nil
}

// Active returns the currently active version of each rule.
func (e *Engine) Active() []VersionedRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var out []VersionedRule
	for id, ver := range e.active {
		for _, r := range e.rules[id] {
			if r.Version == ver && r.Active {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

// History returns all versions of a rule.
func (e *Engine) History(id string) []VersionedRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]VersionedRule{}, e.rules[id]...)
}

// FromFeedback creates a new versioned rule from human feedback.
func (e *Engine) FromFeedback(feedback, context, createdBy string) VersionedRule {
	return e.Propose(VersionedRule{
		Description: feedback,
		Condition:   context,
		Action:      "constrain future proposals",
		Source:      "feedback",
		CreatedBy:   createdBy,
	})
}

// ToAgentsRules converts active rules to the simple agents.Rule slice
// for existing Judge code.
func (e *Engine) ToAgentsRules() []agents.Rule {
	active := e.Active()
	out := make([]agents.Rule, len(active))
	for i, r := range active {
		out[i] = agents.Rule{
			ID:          r.ID,
			Description: r.Description,
			Condition:   r.Condition,
			Action:      r.Action,
			Source:      r.Source,
		}
	}
	return out
}

func (e *Engine) Summary() string {
	active := e.Active()
	if len(active) == 0 {
		return "no active rules"
	}
	s := fmt.Sprintf("%d active rules:\n", len(active))
	for _, r := range active {
		s += fmt.Sprintf("- [%s v%d] %s\n", r.Source, r.Version, r.Description)
	}
	return s
}
