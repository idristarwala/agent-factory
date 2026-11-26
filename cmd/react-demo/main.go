package main

import (
	"context"
	"fmt"
	"os"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/tools"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func main() {
	gw, err := models.NewGatewayFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gateway: %v\n", err)
		os.Exit(1)
	}

	g := graph.NewMemoryGraph()
	g.AddNode(graph.Node{
		ID: "sku-1001", Type: "product",
		Props: map[string]any{"title": "Blue Shirt", "description": "A shirt", "color": "blue"},
	})

	// Register tools
	tr := tools.NewRegistry()
	tr.Register(&tools.MCPTool{
		ServerName: "catalog",
		ToolName:   "lookup_brand",
		Desc:       "Look up brand guidelines",
		CallFn: func(ctx context.Context, args map[string]any) (string, error) {
			return "Brand: Nike. Preferred title format: Nike [Gender] [Color] [Type] [Size]", nil
		},
	})
	tr.Register(&tools.BigQueryTool{Project: "demo", Dataset: "catalog"})

	store := unitofwork.NewMemoryStore()
	steward := agents.NewSmartSteward(store)
	worker := agents.NewReactWorker("react-1", "catalog-enricher", gw, g, tr)

	uows, err := steward.InterpretGoal("Enrich title using brand guidelines", nil)
	if err != nil || len(uows) == 0 {
		fmt.Fprintf(os.Stderr, "steward: %v\n", err)
		os.Exit(1)
	}
	uow := uows[0]
	uow.InputSubgraph = "sku-1001"

	fmt.Println("Running ReAct worker...")
	proposal, err := worker.Execute(uow)
	if err != nil {
		fmt.Fprintf(os.Stderr, "worker: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Proposal: %+v\n", proposal.Updates[0].Props)
	fmt.Printf("Evidence: %v\n", proposal.Evidence)
}
