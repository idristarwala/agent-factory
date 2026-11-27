package agents

import (
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
	"testing"
)

type limitsGateway struct{ calls []models.Request }

func (g *limitsGateway) Generate(req models.Request) (*models.Response, error) {
	g.calls = append(g.calls, req)
	return &models.Response{Content: "invalid"}, nil
}
func TestPinnedAgentLimits(t *testing.T) {
	reg := NewRegistry()
	workerDef, err := reg.Save(AgentVersion{Name: "test", Role: "worker", Config: map[string]string{"max_attempts": "1", "max_output_tokens": "800"}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	invalid := workerDef
	invalid.Config = map[string]string{"max_attempts": "0"}
	if _, err := reg.Save(invalid, 1); err == nil {
		t.Fatal("invalid worker limit accepted")
	}
	gw := &limitsGateway{}
	g := graph.NewMemoryGraph()
	g.AddNode(graph.Node{ID: "case", Props: map[string]any{"status": "open"}})
	worker := NewLLMWorker("agent", "worker", gw, g, nil)
	worker.Definition = workerDef
	if _, err := worker.Execute(&unitofwork.UnitOfWork{ID: "unit", InputSubgraph: "case", AllowedFields: []string{"status"}}); err == nil || len(gw.calls) != 1 || gw.calls[0].MaxTokens != 800 {
		t.Fatal(err, gw.calls)
	}
	judge := NewLLMJudge(gw, g)
	judge.Definition = AgentVersion{Role: "judge", Config: map[string]string{"max_output_tokens": "700"}}
	decision, err := judge.Evaluate(&graph.Mutation{}, nil)
	if err != nil || decision.Approved || len(gw.calls) != 2 || gw.calls[1].MaxTokens != 700 {
		t.Fatal(decision, err, gw.calls)
	}
}
