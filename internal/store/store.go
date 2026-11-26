package store

import (
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// Store is the persistence abstraction.
// Implementations: Memory (in-process) and SQL (any database/sql driver).
// Domain-agnostic: graph, units of work, versioned rules, memory/learnings.
type Store interface {
	// Graph
	GetNode(id graph.NodeID) (*graph.Node, error)
	UpsertNode(n graph.Node) error
	ProposeMutation(m *graph.Mutation) error

	// Unit of work
	CreateUnit(u *unitofwork.UnitOfWork) error
	GetUnit(id string) (*unitofwork.UnitOfWork, error)
	UpdateUnitStatus(id string, status unitofwork.Status) error
	AttachProposal(id string, m *graph.Mutation) error

	// Rules (versioned)
	SaveRuleVersion(id string, version int, data []byte) error
	ActiveRules() ([][]byte, error)

	// Memory / learnings
	SaveEntry(kind, content, source string, meta []byte) error
	EntriesByKind(kind string) ([][]byte, error)

	Close() error
}

// Mode selects the backend.
type Mode string

const (
	ModeMemory Mode = "memory"
	ModeSQL    Mode = "sql"
)
