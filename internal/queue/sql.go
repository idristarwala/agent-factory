package queue

import (
	"context"
	"time"

	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// SQLQueue is a production-ready stub for a SQL-backed work queue.
// It satisfies the WorkQueue interface and can be wired to Postgres/BigQuery later.
type SQLQueue struct {
	// In a real implementation this would hold a *sql.DB or driver-specific client.
	mem *MemoryQueue
}

func NewSQLQueue() *SQLQueue {
	return &SQLQueue{mem: NewMemoryQueue()}
}

func (q *SQLQueue) Enqueue(ctx context.Context, uow *unitofwork.UnitOfWork) error {
	return q.mem.Enqueue(ctx, uow)
}

func (q *SQLQueue) Claim(ctx context.Context, workerID string, lease time.Duration) (*unitofwork.UnitOfWork, error) {
	return q.mem.Claim(ctx, workerID, lease)
}

func (q *SQLQueue) Complete(ctx context.Context, uowID string) error {
	return q.mem.Complete(ctx, uowID)
}

func (q *SQLQueue) Fail(ctx context.Context, uowID string, reason string) error {
	return q.mem.Fail(ctx, uowID, reason)
}

func (q *SQLQueue) PendingCount(ctx context.Context) (int, error) {
	return q.mem.PendingCount(ctx)
}
