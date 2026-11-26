package store

import (
	"fmt"
	"sync"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// MemoryStore is a fully in-process implementation of Store.
// No external dependencies. Suitable for tests and local runs.
type MemoryStore struct {
	mu    sync.RWMutex
	nodes map[graph.NodeID]*graph.Node
	units map[string]*unitofwork.UnitOfWork
	rules map[string][][]byte // id -> versions
	entries []struct {
		Kind, Content, Source string
		Meta                  []byte
	}
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nodes: make(map[graph.NodeID]*graph.Node),
		units: make(map[string]*unitofwork.UnitOfWork),
		rules: make(map[string][][]byte),
	}
}

func (s *MemoryStore) GetNode(id graph.NodeID) (*graph.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", id)
	}
	return n, nil
}

func (s *MemoryStore) UpsertNode(n graph.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[n.ID] = &n
	return nil
}

func (s *MemoryStore) ProposeMutation(m *graph.Mutation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range m.Updates {
		s.nodes[n.ID] = &n
	}
	for _, n := range m.Adds {
		s.nodes[n.ID] = &n
	}
	for _, id := range m.Deletes {
		delete(s.nodes, id)
	}
	return nil
}

func (s *MemoryStore) CreateUnit(u *unitofwork.UnitOfWork) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.units[u.ID]; exists {
		return fmt.Errorf("unit exists: %s", u.ID)
	}
	s.units[u.ID] = u
	return nil
}

func (s *MemoryStore) GetUnit(id string) (*unitofwork.UnitOfWork, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.units[id]
	if !ok {
		return nil, fmt.Errorf("unit not found: %s", id)
	}
	return u, nil
}

func (s *MemoryStore) UpdateUnitStatus(id string, status unitofwork.Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.units[id]
	if !ok {
		return fmt.Errorf("unit not found: %s", id)
	}
	u.Status = status
	return nil
}

func (s *MemoryStore) AttachProposal(id string, m *graph.Mutation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.units[id]
	if !ok {
		return fmt.Errorf("unit not found: %s", id)
	}
	u.Proposal = m
	return nil
}

func (s *MemoryStore) SaveRuleVersion(id string, version int, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules[id] = append(s.rules[id], data)
	return nil
}

func (s *MemoryStore) ActiveRules() ([][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out [][]byte
	for _, versions := range s.rules {
		if len(versions) > 0 {
			out = append(out, versions[len(versions)-1])
		}
	}
	return out, nil
}

func (s *MemoryStore) SaveEntry(kind, content, source string, meta []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, struct {
		Kind, Content, Source string
		Meta                  []byte
	}{kind, content, source, meta})
	return nil
}

func (s *MemoryStore) EntriesByKind(kind string) ([][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out [][]byte
	for _, e := range s.entries {
		if e.Kind == kind {
			out = append(out, []byte(e.Content))
		}
	}
	return out, nil
}

func (s *MemoryStore) Close() error { return nil }
