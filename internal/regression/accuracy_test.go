package regression

import (
	"testing"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
)

// TestWorkerReceivesRules ensures the worker constructor accepts rules
// (regression for issue #10 item 2).
func TestWorkerReceivesRules(t *testing.T) {
	rules := []agents.Rule{{
		Description: "Keep original wording where possible",
	}}
	// We only test construction; full execution requires a gateway.
	w := agents.NewLLMWorker("w1", "worker", nil, graph.NewMemoryGraph(), rules)
	if w == nil {
		t.Fatal("worker is nil")
	}
	if len(w.Rules) != 1 {
		t.Error("rules were not stored on the worker")
	}
}

// TestJudgeConstructorAcceptsGraph ensures the judge can receive graph context
// (regression for issue #10 item 3).
func TestJudgeConstructorAcceptsGraph(t *testing.T) {
	j := agents.NewLLMJudge(nil, graph.NewMemoryGraph())
	if j == nil {
		t.Fatal("judge is nil")
	}
}
