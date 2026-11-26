package graph

import (
	"fmt"
	"sync"
)

// MemoryGraph is a simple in-memory implementation of the central graph.
// Suitable for local development, tests, and early scaffolding.
// Production will use a partitioned / sharded store.

type MemoryGraph struct {
	mu    sync.RWMutex
	nodes map[NodeID]*Node
	edges map[EdgeID]*Edge
}

func NewMemoryGraph() *MemoryGraph {
	return &MemoryGraph{
		nodes: make(map[NodeID]*Node),
		edges: make(map[EdgeID]*Edge),
	}
}

func (g *MemoryGraph) GetNode(id NodeID) (*Node, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", id)
	}
	return n, nil
}

func (g *MemoryGraph) GetSubgraph(root NodeID, depth int) ([]Node, []Edge, error) {
	// Minimal implementation: return the root node only for now.
	// Full BFS traversal will be added later.
	n, err := g.GetNode(root)
	if err != nil {
		return nil, nil, err
	}
	return []Node{*n}, nil, nil
}

func (g *MemoryGraph) ProposeMutation(proposal *Mutation) error {
	// In the real fabric, this would be queued for Judge evaluation.
	// For the in-memory stub we apply immediately so tests can proceed.
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, n := range proposal.Adds {
		g.nodes[n.ID] = &n
	}
	for _, n := range proposal.Updates {
		g.nodes[n.ID] = &n
	}
	for _, e := range proposal.Edges {
		g.edges[e.ID] = &e
	}
	for _, id := range proposal.Deletes {
		delete(g.nodes, id)
	}
	return nil
}

// AddNode is a convenience helper for seeding the graph in tests.
func (g *MemoryGraph) AddNode(n Node) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes[n.ID] = &n
}
