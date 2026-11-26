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

// Apply merges updates into existing nodes so attributes are not lost.
func (g *MemoryGraph) Apply(proposal *Mutation) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, n := range proposal.Adds {
		nCopy := n
		g.nodes[n.ID] = &nCopy
	}
	for _, update := range proposal.Updates {
		existing, ok := g.nodes[update.ID]
		if !ok {
			nCopy := update
			g.nodes[update.ID] = &nCopy
			continue
		}
		// Merge props
		if existing.Props == nil {
			existing.Props = make(map[string]any)
		}
		for k, v := range update.Props {
			existing.Props[k] = v
		}
		if update.Type != "" {
			existing.Type = update.Type
		}
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
		// Deep-ish copy to avoid mutation through snapshot
		nCopy := *n
		if n.Props != nil {
			nCopy.Props = make(map[string]any, len(n.Props))
			for k, v := range n.Props {
				nCopy.Props[k] = v
			}
		}
		snap[id] = &nCopy
	}
	return snap
}

func (g *MemoryGraph) AddNode(n Node) {
	g.mu.Lock()
	defer g.mu.Unlock()
	nCopy := n
	g.nodes[n.ID] = &nCopy
}
