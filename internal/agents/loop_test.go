package agents_test

import (
	"testing"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func TestProposalJudgeLoop(t *testing.T) {
	store := unitofwork.NewMemoryStore()
	g := graph.NewMemoryGraph()
	steward := agents.NewStubSteward(store)
	judge := agents.NewStubJudge()

	// Seed a node
	g.AddNode(graph.Node{ID: "product-1", Type: "product", Props: map[string]any{"title": "old title"}})

	// Steward turns goal into unit of work
	uows, err := steward.InterpretGoal("enrich title for product-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(uows) != 1 {
		t.Fatalf("expected 1 unit of work, got %d", len(uows))
	}

	uow := uows[0]

	// Simulate worker proposing a mutation
	proposal := &graph.Mutation{
		UnitOfWorkID: uow.ID,
		Updates: []graph.Node{
			{ID: "product-1", Type: "product", Props: map[string]any{"title": "new enriched title"}},
		},
		Evidence: []string{"source: catalog-agent"},
	}

	if err := store.AttachProposal(uow.ID, proposal); err != nil {
		t.Fatal(err)
	}

	// Judge evaluates
	decision, err := judge.Evaluate(proposal, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Approved {
		t.Fatal("expected approval from stub judge")
	}

	// Apply to graph
	if err := g.ProposeMutation(proposal); err != nil {
		t.Fatal(err)
	}

	n, err := g.GetNode("product-1")
	if err != nil {
		t.Fatal(err)
	}
	if n.Props["title"] != "new enriched title" {
		t.Fatalf("title not updated: %v", n.Props["title"])
	}
}
