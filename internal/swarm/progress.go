package swarm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/idristarwala/agent-factory/internal/graph"
)

// Progress tracks whether the graph is still changing across rounds.
type Progress struct {
	LastHash     string
	Stagnant     int
	MaxStagnant  int
	UnitFailures map[string]int
	MaxUnitFails int
}

func NewProgress() *Progress {
	return &Progress{
		MaxStagnant:  2,
		UnitFailures: make(map[string]int),
		MaxUnitFails: 3,
	}
}

// Observe updates the progress state from the current graph snapshot.
// Returns true if the swarm should stop early (no progress).
func (p *Progress) Observe(snap any) bool {
	h := hashSnapshot(snap)
	if h == p.LastHash {
		p.Stagnant++
	} else {
		p.Stagnant = 0
		p.LastHash = h
	}
	return p.Stagnant >= p.MaxStagnant
}

// RecordFailure increments the failure count for a unit.
// Returns true if the unit should be marked blocked.
func (p *Progress) RecordFailure(unitID string) bool {
	p.UnitFailures[unitID]++
	return p.UnitFailures[unitID] >= p.MaxUnitFails
}

func hashSnapshot(snap any) string {
	b, _ := json.Marshal(snap)
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum[:8])
}

// BlockedNodes returns node IDs that have failed too many times.
func (p *Progress) BlockedNodes() []graph.NodeID {
	var out []graph.NodeID
	for id, n := range p.UnitFailures {
		if n >= p.MaxUnitFails {
			out = append(out, graph.NodeID(id))
		}
	}
	return out
}
