package graph

import "github.com/idristarwala/agent-factory/internal/llm"

// Central Graph is the single source of truth.
// All agents propose mutations; Judge evaluates them before commit.

type NodeID string
type EdgeID string

type Node struct {
	ID    NodeID
	Type  string
	Props map[string]any
}

type Edge struct {
	ID    EdgeID
	From  NodeID
	To    NodeID
	Type  string
	Props map[string]any
}

type Graph interface {
	GetNode(id NodeID) (*Node, error)
	GetSubgraph(root NodeID, depth int) ([]Node, []Edge, error)
	ProposeMutation(proposal *Mutation) error
	Apply(proposal *Mutation) error
	Snapshot() any
}

type Mutation struct {
	AgentName     string         `json:",omitempty"`
	AgentRevision int            `json:",omitempty"`
	RuleRevisions map[string]int `json:",omitempty"`
	UnitOfWorkID  string
	Adds          []Node
	Updates       []Node
	Edges         []Edge
	Deletes       []NodeID
	Evidence      []string
	Images        []llm.ImagePart `json:"-"`
}
