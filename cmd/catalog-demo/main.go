package main

import (
	"fmt"
	factoryStore "github.com/idristarwala/agent-factory/internal/store"
	"log"
	"os"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/configuredgateway"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func main() {
	database, err := factoryStore.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	definitions := factoryStore.FactoryDefinitions(database, "catalog-demo")
	registry := agents.NewStoredRegistry(definitions)

	gw, err := configuredgateway.NewFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gateway: %v\n", err)
		os.Exit(1)
	}

	g := graph.NewMemoryGraph()
	store := unitofwork.NewMemoryStore()
	ruleEngine := rules.NewStoredEngine(definitions)

	g.AddNode(graph.Node{
		ID:   "sku-1001",
		Type: "product",
		Props: map[string]any{
			"title":       "Blue Shirt",
			"description": "A shirt",
		},
	})

	ruleEngine.Propose(rules.VersionedRule{
		Description: "Titles must be descriptive and include color and product type",
		Source:      "human",
		CreatedBy:   "operator",
	})

	if _, err := registry.Resolve(agents.AgentVersion{Name: "steward", Role: "steward", ModelTier: "frontier", CreatedBy: "system"}); err != nil {
		log.Fatal(err)
	}
	steward := agents.NewStubSteward(store)
	steward.Registry = registry
	activeRules := ruleEngine.ToAgentsRules()
	steward.SetDependencies(gw, g)
	def, err := registry.Resolve(agents.AgentVersion{Name: "worker-1", Role: "catalog-enricher", ModelTier: "cheap", CreatedBy: "steward"})
	if err != nil {
		log.Fatal(err)
	}
	worker := agents.NewLLMWorker(def.Name, def.Role, gw, g, activeRules)
	worker.Definition = def
	if _, err := registry.Resolve(agents.AgentVersion{Name: "judge", Role: "judge", ModelTier: "frontier", CreatedBy: "system"}); err != nil {
		log.Fatal(err)
	}
	judge := agents.NewLLMJudge(gw, g)

	uows, err := steward.InterpretGoal("Enrich title and description for sku-1001", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "steward: %v\n", err)
		os.Exit(1)
	}
	uow := uows[0]
	uow.InputSubgraph = "sku-1001"
	fmt.Printf("Created unit of work: %s\n", uow.ID)

	proposal, err := worker.Execute(uow)
	if err != nil {
		fmt.Fprintf(os.Stderr, "worker: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Worker proposal: %+v\n", proposal.Updates[0].Props)

	decision, err := judge.Evaluate(proposal, ruleEngine.ToAgentsRules())
	if err != nil {
		fmt.Fprintf(os.Stderr, "judge: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Judge decision: approved=%v reason=%s\n", decision.Approved, decision.Reason)

	if decision.Approved {
		if err := g.ProposeMutation(proposal); err != nil {
			fmt.Fprintf(os.Stderr, "apply: %v\n", err)
			os.Exit(1)
		}
		n, _ := g.GetNode("sku-1001")
		fmt.Printf("Graph updated: title=%v description=%v\n", n.Props["title"], n.Props["description"])
	}

	fb := ruleEngine.FromFeedback("Always mention material if known", "catalog titles", "operator")
	fmt.Printf("Feedback turned into rule: %s\n", fb.Description)
	fmt.Printf("Current rules:\n%s", ruleEngine.Summary())
}
