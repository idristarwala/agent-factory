package unitofwork

import (
	"fmt"
	"sync"

	"github.com/idristarwala/agent-factory/internal/graph"
)

// MemoryStore is a simple in-memory unit-of-work store.

type MemoryStore struct {
	mu    sync.RWMutex
	units map[string]*UnitOfWork
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		units: make(map[string]*UnitOfWork),
	}
}

func (s *MemoryStore) Create(u *UnitOfWork) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.units[u.ID]; exists {
		return fmt.Errorf("unit of work already exists: %s", u.ID)
	}
	u.Status = StatusProposed
	s.units[u.ID] = u
	return nil
}

func (s *MemoryStore) Claim(id string, agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.units[id]
	if !ok {
		return fmt.Errorf("unit of work not found: %s", id)
	}
	if u.Status != StatusProposed {
		return fmt.Errorf("unit of work not claimable: %s", u.Status)
	}
	u.Status = StatusClaimed
	u.ClaimedBy = agentID
	return nil
}

func (s *MemoryStore) UpdateStatus(id string, status Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.units[id]
	if !ok {
		return fmt.Errorf("unit of work not found: %s", id)
	}
	u.Status = status
	return nil
}

func (s *MemoryStore) AttachProposal(id string, m *graph.Mutation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.units[id]
	if !ok {
		return fmt.Errorf("unit of work not found: %s", id)
	}
	u.Proposal = m
	u.Status = StatusReviewed
	return nil
}

func (s *MemoryStore) Get(id string) (*UnitOfWork, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.units[id]
	if !ok {
		return nil, fmt.Errorf("unit of work not found: %s", id)
	}
	return u, nil
}
