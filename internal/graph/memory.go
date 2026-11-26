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

// Apply merges updates into existing nodes instead of replacing them.
// This prevents attribute loss when a worker only updates title/description.
func (g *MemoryGraph) Apply(proposal *Mutation) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, n := range proposal.Adds {
		nCopy := n
		if nCopy.Props == nil {
			nCopy.Props = make(map[string]any)
		}
		g.nodes[n.ID] = &nCopy
	}

	for _, n := range proposal.Updates {
		existing, ok := g.nodes[n.ID]
		if !ok {
			// Treat as add if node does not exist
			nCopy := n
			if nCopy.Props == nil {
				nCopy.Props = make(map[string]any)
			}
			g.nodes[n.ID] = &nCopy
			continue
		}
		// Merge props: new values overwrite, existing values are preserved
		if existing.Props == nil {
			existing.Props = make(map[string]any)
		}
		for k, v := range n.Props {
			existing.Props[k] = v
		}
		if n.Type != "" {
			existing.Type = n.Type
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
		// Deep copy the node to prevent external mutation
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
	if nCopy.Props == nil {
		nCopy.Props = make(map[string]any)
	}
	g.nodes[n.ID] = &nCopy
}
