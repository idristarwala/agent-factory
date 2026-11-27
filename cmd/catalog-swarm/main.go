package main

import (
	"context"
	"fmt"
	factoryStore "github.com/idristarwala/agent-factory/internal/store"
	"github.com/idristarwala/agent-factory/internal/tools"
	"log"
	"os"
	"time"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/catalog"
	"github.com/idristarwala/agent-factory/internal/configuredgateway"
	"github.com/idristarwala/agent-factory/internal/domain"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/memory"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func needsWork(n graph.Node) bool {
	title, _ := n.Props["title"].(string)
	desc, _ := n.Props["description"].(string)
	if len(title) < 15 {
		return true
	}
	if len(desc) < 20 {
		return true
	}
	if mat, ok := n.Props["material"].(string); ok && mat == "" && (n.Props["pt"] == "Apparel" || n.Props["pt"] == "Footwear") {
		return true
	}
	return false
}

func main() {
	var evidenceConfig *tools.EvidenceConfig
	if path := os.Getenv("FACTORY_EVIDENCE_CONFIG"); path != "" {
		config, err := tools.LoadEvidenceConfig(path, domain.CatalogPack().Name)
		if err != nil {
			log.Fatal(err)
		}
		evidenceConfig = config
	}
	database, err := factoryStore.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	definitions := factoryStore.FactoryDefinitions(database, "catalog-swarm")
	registry := agents.NewStoredRegistry(definitions)

	gw, err := configuredgateway.NewFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gateway: %v\n", err)
		os.Exit(1)
	}

	g := graph.NewMemoryGraph()
	for _, n := range catalog.Seed() {
		g.AddNode(n)
	}

	store := unitofwork.NewMemoryStore()
	ruleEngine := rules.NewStoredEngine(definitions)
	mem := memory.NewStore()
	agentReg := registry

	for i, r := range domain.CatalogPack().DefaultRules {
		if err := ruleEngine.Install(rules.VersionedRule{ID: fmt.Sprintf("catalog/default/%d", i+1), Description: r, Active: true, Source: "domain-pack", CreatedBy: "system"}); err != nil {
			log.Fatal(err)
		}
	}
	ruleEngine.Propose(rules.VersionedRule{
		Description: "Titles should include brand and key attributes. Descriptions should be specific.",
		Source:      "human",
		CreatedBy:   "operator",
	})

	fmt.Println("=== Agent Factory (domain-agnostic core, catalog pack) ===")
	fmt.Println(agentReg.Summary())
	fmt.Println(ruleEngine.Summary())
	fmt.Println("Variant groups:")
	for group, ids := range catalog.VariantGroups() {
		fmt.Printf("  %s: %v\n", group, ids)
	}
	fmt.Println()

	if _, err := registry.Resolve(agents.AgentVersion{Name: "steward", Role: "steward", ModelTier: "frontier", CreatedBy: "system"}); err != nil {
		log.Fatal(err)
	}
	steward := agents.NewSmartSteward(store)
	steward.Registry = registry
	activeRules := ruleEngine.ToAgentsRules()
	steward.SetRuntime(gw, g, activeRules)
	def, err := registry.Resolve(agents.AgentVersion{Name: "worker-1", Role: "catalog-corrector", ModelTier: "cheap", CreatedBy: "steward"})
	if err != nil {
		log.Fatal(err)
	}
	worker := agents.NewLLMWorker(def.Name, def.Role, gw, g, activeRules)
	worker.Definition = def
	if _, err := registry.Resolve(agents.AgentVersion{Name: "judge", Role: "judge", ModelTier: "frontier", CreatedBy: "system"}); err != nil {
		log.Fatal(err)
	}
	judge := agents.NewLLMJudge(gw, g)

	nodes := catalog.Seed()
	corrected := 0
	for _, n := range nodes {
		if !needsWork(n) {
			fmt.Printf("SKIP: %s — %v\n", n.ID, n.Props["title"])
			continue
		}

		fmt.Printf("\n--- %s (PT=%v Brand=%v) ---\n", n.ID, n.Props["pt"], n.Props["brand"])
		fmt.Printf("  Before: %q\n", n.Props["title"])

		uows, err := steward.InterpretGoal(fmt.Sprintf("Improve title and description for %s", n.ID), nil)
		if err != nil || len(uows) == 0 {
			fmt.Printf("  Steward error: %v\n", err)
			continue
		}
		uow := uows[0]
		uow.InputSubgraph = n.ID

		if evidenceConfig != nil {
			evidence, err := evidenceConfig.Gather(context.Background(), string(n.ID), n.Props)
			if err != nil {
				log.Fatalf("gather evidence for %s: %v", n.ID, err)
			}
			uow.Evidence = append(uow.Evidence, evidence...)
		}
		proposal, err := worker.Execute(uow)
		if err != nil {
			fmt.Printf("  Worker error: %v\n", err)
			continue
		}

		proposal.Evidence = append(proposal.Evidence, uow.Evidence...)
		decision, err := judge.Evaluate(proposal, ruleEngine.ToAgentsRules())
		if err != nil {
			fmt.Printf("  Judge error: %v\n", err)
			continue
		}

		if decision.Approved && len(proposal.Updates) > 0 {
			g.ProposeMutation(proposal)
			updated, _ := g.GetNode(n.ID)
			fmt.Printf("  After:  %q\n", updated.Props["title"])
			fmt.Printf("  Judge: %s\n", decision.Reason)
			mem.Add("learning", fmt.Sprintf("%s -> %v", n.ID, updated.Props["title"]), "swarm", nil)
			corrected++
		} else {
			fmt.Printf("  Rejected: %s\n", decision.Reason)
		}
	}

	fmt.Printf("\n=== Done at %s ===\nCorrected: %d\n%s\n", time.Now().Format(time.RFC3339), corrected, ruleEngine.Summary())
}
