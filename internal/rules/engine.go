package rules

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/agents"
)

// Engine is a simple in-memory rule store.
// Human feedback and agent proposals become rules here.

type Engine struct {
	mu    sync.RWMutex
	rules []agents.Rule
}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Add(r agents.Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	e.rules = append(e.rules, r)
}

func (e *Engine) All() []agents.Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]agents.Rule, len(e.rules))
	copy(out, e.rules)
	return out
}

// FromFeedback turns human feedback on a result into a rule.
func (e *Engine) FromFeedback(feedback string, context string) agents.Rule {
	r := agents.Rule{
		ID:          uuid.NewString(),
		Description: feedback,
		Condition:   context,
		Action:      "constrain future proposals",
		Source:      "feedback",
	}
	e.Add(r)
	return r
}

func (e *Engine) Summary() string {
	rules := e.All()
	if len(rules) == 0 {
		return "no rules"
	}
	s := fmt.Sprintf("%d rules:\n", len(rules))
	for _, r := range rules {
		s += fmt.Sprintf("- [%s] %s\n", r.Source, r.Description)
	}
	return s
}
