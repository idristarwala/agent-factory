package agents

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
	ID          string
	Description string
	Condition   string
	Action      string
	Source      string // "human", "feedback", "agent"
}
