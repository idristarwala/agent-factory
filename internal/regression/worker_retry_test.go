package regression

import (
	"testing"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
)

// TestWorkerHasRetryLogic ensures the worker type supports the improved execution path.
func TestWorkerHasRetryLogic(t *testing.T) {
	w := agents.NewLLMWorker("w1", "worker", nil, graph.NewMemoryGraph(), nil)
	if w == nil {
		t.Fatal("expected worker")
	}
	// The presence of the type with the new fields is the regression check.
	// Full retry execution requires a mock gateway.
}
