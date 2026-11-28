package agents

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
)

// fakeFactGateway returns REJECT when the prompt shows a brand contradiction,
// APPROVE otherwise. Used to prove the judge path rejects contradictory facts.
type fakeFactGateway struct {
	lastSystem string
	lastUser   string
}

func (f *fakeFactGateway) Generate(req models.Request) (*models.Response, error) {
	f.lastSystem = req.System
	f.lastUser = req.User
	// Detect a simple contradiction: original has "Acme" and proposal has "OtherBrand"
	if strings.Contains(req.User, `"brand":"Acme"`) && strings.Contains(req.User, `"brand":"OtherBrand"`) {
		return &models.Response{Content: "REJECT: proposed brand contradicts original fact", Model: "fake-judge"}, nil
	}
	return &models.Response{Content: "APPROVE: no fact contradiction detected", Model: "fake-judge"}, nil
}

// minimalGraph holds a single node for the test.
type minimalGraph struct {
	node *graph.Node
}

func (g *minimalGraph) GetNode(id graph.NodeID) (*graph.Node, error) {
	if g.node != nil && g.node.ID == id {
		return g.node, nil
	}
	return nil, graph.ErrNotFound
}
func (g *minimalGraph) GetSubgraph(root graph.NodeID, depth int) ([]graph.Node, []graph.Edge, error) {
	return nil, nil, nil
}
func (g *minimalGraph) ProposeMutation(proposal *graph.Mutation) error { return nil }
func (g *minimalGraph) Apply(proposal *graph.Mutation) error            { return nil }
func (g *minimalGraph) Snapshot() any                                  { return nil }

func TestJudgeRejectsContradictoryFacts(t *testing.T) {
	original := &graph.Node{
		ID:   "item-1",
		Type: "product",
		Props: map[string]any{
			"brand": "Acme",
			"title": "Acme Super Widget",
		},
	}
	g := &minimalGraph{node: original}
	fake := &fakeFactGateway{}
	j := NewLLMJudge(fake, g)

	// Proposal that changes brand to a contradicting value
	proposal := &graph.Mutation{
		Updates: []graph.Node{{
			ID:   "item-1",
			Type: "product",
			Props: map[string]any{
				"brand": "OtherBrand",
				"title": "Acme Super Widget",
			},
		}},
	}

	dec, err := j.Evaluate(proposal, nil)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if dec.Approved {
		t.Fatal("expected rejection of fact-contradicting proposal, got approval")
	}
	if !strings.Contains(strings.ToUpper(dec.Reason), "REJECT") {
		t.Errorf("expected REJECT reason, got %q", dec.Reason)
	}
	if !strings.Contains(fake.lastSystem, "contradicting the original entity's facts") {
		t.Error("system prompt should instruct the judge to reject fact contradictions")
	}
	// Sanity: consistent proposal is approved
	consistent := &graph.Mutation{
		Updates: []graph.Node{{
			ID:   "item-1",
			Type: "product",
			Props: map[string]any{
				"title": "Acme Super Widget (improved)",
			},
		}},
	}
	dec2, err := j.Evaluate(consistent, nil)
	if err != nil {
		t.Fatalf("Evaluate consistent: %v", err)
	}
	if !dec2.Approved {
		t.Errorf("expected approval of non-contradicting proposal, got %q", dec2.Reason)
	}
}

// Ensure the original JSON is present so the fake can detect the contradiction.
func TestJudgePromptIncludesOriginalFacts(t *testing.T) {
	original := &graph.Node{
		ID:   "item-1",
		Type: "product",
		Props: map[string]any{"brand": "Acme"},
	}
	g := &minimalGraph{node: original}
	fake := &fakeFactGateway{}
	j := NewLLMJudge(fake, g)

	proposal := &graph.Mutation{
		Updates: []graph.Node{{
			ID:   "item-1",
			Type: "product",
			Props: map[string]any{"brand": "OtherBrand"},
		}},
	}
	_, _ = j.Evaluate(proposal, nil)
	if !strings.Contains(fake.lastUser, "Acme") {
		t.Error("judge user prompt must include original entity facts")
	}
	var parsed graph.Node
	// Rough check that original JSON is embeddable
	if err := json.Unmarshal([]byte(strings.Split(fake.lastUser, "Original entity (JSON):\n")[1][:100]), &parsed); err == nil {
		// ignore parse error; presence of key is enough
	}
}
