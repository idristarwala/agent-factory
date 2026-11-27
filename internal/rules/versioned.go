package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/agents"
)

// VersionedRule is an immutable rule version.
type VersionedRule struct {
	ID            string
	Version       int
	Description   string
	Condition     string
	Action        string
	Source        string
	Active        bool
	CreatedAt     time.Time
	CreatedBy     string
	ParentID      string
	ActivatedFrom int
}

// Engine is a versioned rule store.
type Engine struct{ store revision.Store }

func NewEngine() *Engine                        { return NewStoredEngine(revision.NewMemory()) }
func NewStoredEngine(st revision.Store) *Engine { return &Engine{store: st} }

func (e *Engine) List(id string) ([]VersionedRule, error) {
	rows, err := e.store.Read(revision.Rule, id)
	if err != nil {
		return nil, err
	}
	result := make([]VersionedRule, 0, len(rows))
	for _, row := range rows {
		var r VersionedRule
		// The original feedback API persisted only the raw note, even when it contained JSON.
		if json.Unmarshal(row.Data, &r) != nil || r.ID != row.ID || r.Version != row.Version {
			r = VersionedRule{ID: row.ID, Version: row.Version, Description: string(row.Data), Source: "feedback", CreatedBy: "customer", Active: true}
		}
		result = append(result, r)
	}
	return result, nil
}
func (e *Engine) Save(r VersionedRule, expected int) (VersionedRule, error) {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	r.Version = expected + 1
	r.CreatedAt = time.Now().UTC()
	r.ParentID = ""
	if expected > 0 {
		r.ParentID = fmt.Sprintf("%s:%d", r.ID, expected)
	}
	data, err := json.Marshal(r)
	if err != nil {
		return VersionedRule{}, err
	}
	err = e.store.Insert(revision.Rule, revision.Snapshot{ID: r.ID, Version: r.Version, Data: data})
	if err != nil {
		return VersionedRule{}, err
	}
	return r, nil
}

// Propose is the in-process runtime entry point. Competing proposals each become a revision.
func (e *Engine) Propose(r VersionedRule) VersionedRule {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	for {
		h := e.History(r.ID)
		expected := 0
		if len(h) > 0 {
			expected = h[len(h)-1].Version
		}
		v, err := e.Save(r, expected)
		if errors.Is(err, revision.ErrConflict) {
			continue
		}
		if err != nil {
			panic(err)
		}
		return v
	}
}
func (e *Engine) Promote(id string, version, expected int, actor string) (VersionedRule, error) {
	h, err := e.List(id)
	if err != nil {
		return VersionedRule{}, err
	}
	if len(h) == 0 {
		return VersionedRule{}, fmt.Errorf("unknown rule %q", id)
	}
	if h[len(h)-1].Version != expected {
		return VersionedRule{}, revision.ErrConflict
	}
	var selected *VersionedRule
	for i := range h {
		if h[i].Version == version {
			selected = &h[i]
		}
	}
	if selected == nil {
		return VersionedRule{}, fmt.Errorf("unknown version %d", version)
	}
	for _, r := range slices.Backward(h) {
		if r.Active {
			if r.Version == version || r.ActivatedFrom == version {
				return r, nil
			}
			break
		}
	}
	next := *selected
	next.Active = true
	next.ActivatedFrom = version
	next.CreatedBy = actor
	return e.Save(next, expected)
}
func (e *Engine) Activate(id string, version int) error {
	h, err := e.List(id)
	if err != nil {
		return err
	}
	if len(h) == 0 {
		return fmt.Errorf("rule %s not found", id)
	}
	_, err = e.Promote(id, version, h[len(h)-1].Version, "system")
	return err
}
func (e *Engine) Current() ([]VersionedRule, error) {
	history, err := e.List("")
	if err != nil {
		return nil, err
	}
	out := []VersionedRule{}
	for _, r := range history {
		if !r.Active {
			continue
		}
		if len(out) > 0 && out[len(out)-1].ID == r.ID {
			out[len(out)-1] = r
		} else {
			out = append(out, r)
		}
	}
	return out, nil
}
func (e *Engine) Active() []VersionedRule {
	out, err := e.Current()
	if err != nil {
		panic(err)
	}
	return out
}
func (e *Engine) History(id string) []VersionedRule {
	out, err := e.List(id)
	if err != nil {
		panic(err)
	}
	return out
}

// FromFeedback creates a new versioned rule from human feedback.
func (e *Engine) FromFeedback(feedback, context, createdBy string) VersionedRule {
	return e.Propose(VersionedRule{
		Description: feedback,
		Condition:   context,
		Action:      "constrain future proposals",
		Source:      "feedback",
		CreatedBy:   createdBy,
		Active:      true,
	})
}

// ToAgentsRules converts active rules to the simple agents.Rule slice.
func (e *Engine) ToAgentsRules() []agents.Rule {
	active := e.Active()
	out := make([]agents.Rule, len(active))
	for i, r := range active {
		out[i] = agents.Rule{
			ID:          r.ID,
			Version:     r.Version,
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
