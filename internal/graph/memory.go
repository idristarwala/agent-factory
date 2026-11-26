package graph

import (
	"fmt"
	"sync"
)

// MemoryGraph is a simple in-memory implementation of the central graph.
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
	n, err := g.GetNode(root)
	if err != nil {
		return nil, nil, err
	}
	return []Node{*n}, nil, nil
}

func (g *MemoryGraph) ProposeMutation(proposal *Mutation) error {
	return g.Apply(proposal)
}

func (g *MemoryGraph) Apply(proposal *Mutation) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, n := range proposal.Adds {
		nCopy := n
		g.nodes[n.ID] = &nCopy
	}
	for _, n := range proposal.Updates {
		nCopy := n
		g.nodes[n.ID] = &nCopy
	}
	for _, e := range proposal.Edges {
		eCopy := e
		g.edges[e.ID] = &eCopy
	}
	for _, id := range proposal.Deletes {
		delete(g.nodes, id)
	}
	return nil
}

func (g *MemoryGraph) Snapshot() any {
	g.mu.RLock()
	defer g.mu.RUnlock()
	snap := make(map[NodeID]*Node, len(g.nodes))
	for id, n := range g.nodes {
		snap[id] = n
	}
	return snap
}

func (g *MemoryGraph) AddNode(n Node) {
	g.mu.Lock()
	defer g.mu.Unlock()
	nCopy := n
	g.nodes[n.ID] = &nCopy
}
