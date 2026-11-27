package agents

import "strings"
import "github.com/idristarwala/agent-factory/internal/graph"
import "github.com/idristarwala/agent-factory/internal/unitofwork"

// Steward interprets goals, creates agents, and assigns units of work.
type Steward interface {
	InterpretGoal(goal string, graphSnapshot any) ([]*unitofwork.UnitOfWork, error)
	CreateWorker(spec WorkerSpec) (Worker, error)
	SetAutonomyMode(mode string) error
}

// Judge (Arbiter) evaluates proposals.
type Judge interface {
	Evaluate(proposal *graph.Mutation, rules []Rule) (Decision, error)
}

type Decision struct {
	Approved bool
	Reason   string
	Feedback string
}

// Worker claims units of work and proposes mutations.
type Worker interface {
	Claim(uow *unitofwork.UnitOfWork) error
	Execute(uow *unitofwork.UnitOfWork) (*graph.Mutation, error)
}

type WorkerSpec struct {
	Name      string
	Role      string
	ModelTier string // "cheap" only for workers
	Tools     []string
}

type Rule struct {
	Version     int
	ID          string
	Description string
	Condition   string
	Action      string
	Source      string // "human", "feedback", "agent"
}

func ruleRevisions(rules []Rule) map[string]int {
	var versions map[string]int
	for _, r := range rules {
		if r.ID != "" && r.Version > 0 {
			if versions == nil {
				versions = make(map[string]int)
			}
			versions[r.ID] = r.Version
		}
	}
	return versions
}

func renderRule(r Rule) string {
	parts := []string{r.Description}
	if r.Condition != "" {
		parts = append(parts, "Condition: "+r.Condition)
	}
	if r.Action != "" {
		parts = append(parts, "Action: "+r.Action)
	}
	return strings.Join(parts, " | ")
}
