package main

import (
	"fmt"
	factoryStore "github.com/idristarwala/agent-factory/internal/store"
	"log"
	"os"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/domain"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/memory"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func main() {
	database, err := factoryStore.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	definitions := factoryStore.FactoryDefinitions(database, "")
	registry := agents.NewStoredRegistry(definitions)

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: goal \"<goal text>\"\n")
		os.Exit(1)
	}
	goal := os.Args[1]

	gw, err := models.NewGatewayFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gateway: %v\n", err)
		os.Exit(1)
	}

	g := graph.NewMemoryGraph()
	store := unitofwork.NewMemoryStore()
	ruleEngine := rules.NewStoredEngine(definitions)
	mem := memory.NewStore()

	g.AddNode(graph.Node{
		ID:   "sku-1001",
		Type: "product",
		Props: map[string]any{
			"title":       "Blue Shirt",
			"description": "A shirt",
			"color":       "blue",
		},
	})

	for i, r := range domain.CatalogPack().DefaultRules {
		if err := ruleEngine.Install(rules.VersionedRule{ID: fmt.Sprintf("catalog/default/%d", i+1), Description: r, Active: true, Source: "domain-pack", CreatedBy: "system"}); err != nil {
			log.Fatal(err)
		}
	}

	if _, err := registry.Resolve(agents.AgentVersion{Name: "steward", Role: "steward", ModelTier: "frontier", CreatedBy: "system"}); err != nil {
		log.Fatal(err)
	}
	steward := agents.NewSmartSteward(store)
	steward.Registry = registry
	activeRules := ruleEngine.ToAgentsRules()
	steward.SetRuntime(gw, g, activeRules)
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

	fmt.Printf("Goal: %s\n", goal)
	uows, err := steward.InterpretGoal(goal, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "steward: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Steward created %d unit(s) of work\n", len(uows))

	for _, uow := range uows {
		uow.InputSubgraph = "sku-1001"
		fmt.Printf("\n--- Processing %s ---\n", uow.ID[:8])

		proposal, err := worker.Execute(uow)
		if err != nil {
			fmt.Printf("Worker error: %v\n", err)
			continue
		}
		fmt.Printf("Proposal: %v\n", proposal.Updates[0].Props)

		decision, err := judge.Evaluate(proposal, ruleEngine.ToAgentsRules())
		if err != nil {
			fmt.Printf("Judge error: %v\n", err)
			continue
		}
		fmt.Printf("Judge: approved=%v reason=%s\n", decision.Approved, decision.Reason)

		if decision.Approved {
			g.ProposeMutation(proposal)
			n, _ := g.GetNode("sku-1001")
			fmt.Printf("Graph now: title=%v\n", n.Props["title"])

			mem.Add("learning", fmt.Sprintf("Enriched title to: %v", n.Props["title"]), "worker", nil)
		}
	}

	fmt.Printf("\nLearnings recorded: %d\n", len(mem.ByKind("learning")))
	fmt.Printf("Rules in engine:\n%s", ruleEngine.Summary())
}
