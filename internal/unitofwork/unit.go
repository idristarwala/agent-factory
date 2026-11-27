package unitofwork

import "github.com/idristarwala/agent-factory/internal/graph"

type Status string

const (
	StatusProposed   Status = "proposed"
	StatusClaimed    Status = "claimed"
	StatusInProgress Status = "in_progress"
	StatusReviewed   Status = "reviewed"
	StatusCommitted  Status = "committed"
	StatusRejected   Status = "rejected"
)

type UnitOfWork struct {
	ID              string
	GoalID          string
	Description     string
	InputSubgraph   graph.NodeID
	ExpectedOutput  string
	Budget          map[string]any
	Mode            string
	Status          Status
	ClaimedBy       string
	Proposal        *graph.Mutation
	Evidence        []string
	AllowedFields   []string // fields the worker may change; schema is built from these
	RequiredFields  []string
}

type Store interface {
	Create(u *UnitOfWork) error
	Claim(id string, agentID string) error
	UpdateStatus(id string, status Status) error
	AttachProposal(id string, m *graph.Mutation) error
}
