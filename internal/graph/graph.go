package graph

// Central Graph is the single source of truth.
// All agents propose mutations; Judge evaluates them before commit.

type NodeID string
type EdgeID string

type Node struct {
	ID     NodeID
	Type   string
	Props  map[string]any
}

type Edge struct {
	ID     EdgeID
	From   NodeID
	To     NodeID
	Type   string
	Props  map[string]any
}

type Graph interface {
	GetNode(id NodeID) (*Node, error)
	GetSubgraph(root NodeID, depth int) ([]Node, []Edge, error)
	ProposeMutation(proposal *Mutation) error
	Apply(m *Mutation) error
	Snapshot() any
}

type Mutation struct {
	UnitOfWorkID string
	Adds         []Node
	Updates      []Node
	Edges        []Edge
	Deletes      []NodeID
	Evidence     []string
}
