package queue

import (
	"context"
	"sync"
	"time"

	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// WorkQueue is a claimable queue for units of work.
type WorkQueue interface {
	Enqueue(ctx context.Context, uow *unitofwork.UnitOfWork) error
	Claim(ctx context.Context, workerID string, lease time.Duration) (*unitofwork.UnitOfWork, error)
	Complete(ctx context.Context, uowID string) error
	Fail(ctx context.Context, uowID string, reason string) error
	PendingCount(ctx context.Context) (int, error)
}

// MemoryQueue is an in-memory implementation suitable for local and tests.
type MemoryQueue struct {
	mu      sync.Mutex
	pending []*unitofwork.UnitOfWork
	leases  map[string]time.Time // uowID -> expiry
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		leases: make(map[string]time.Time),
	}
}

func (q *MemoryQueue) Enqueue(ctx context.Context, uow *unitofwork.UnitOfWork) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending = append(q.pending, uow)
	return nil
}

func (q *MemoryQueue) Claim(ctx context.Context, workerID string, lease time.Duration) (*unitofwork.UnitOfWork, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now()
	// Clean expired leases
	for id, exp := range q.leases {
		if now.After(exp) {
			delete(q.leases, id)
		}
	}
	for i, uow := range q.pending {
		if _, leased := q.leases[uow.ID]; !leased {
			q.leases[uow.ID] = now.Add(lease)
			// Remove from pending for simplicity; in real impl move to in-progress
			q.pending = append(q.pending[:i], q.pending[i+1:]...)
			return uow, nil
		}
	}
	return nil, nil // nothing to claim
}

func (q *MemoryQueue) Complete(ctx context.Context, uowID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.leases, uowID)
	return nil
}

func (q *MemoryQueue) Fail(ctx context.Context, uowID string, reason string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.leases, uowID)
	// Could requeue here
	return nil
}

func (q *MemoryQueue) PendingCount(ctx context.Context) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.pending), nil
}
