package main

import (
	"fmt"
	"os"
	"time"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/catalog"
	"github.com/idristarwala/agent-factory/internal/domain"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/memory"
	"github.com/idristarwala/agent-factory/internal/models"
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
	gw, err := models.NewGatewayFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gateway: %v\n", err)
		os.Exit(1)
	}

	g := graph.NewMemoryGraph()
	for _, n := range catalog.Seed() {
		g.AddNode(n)
	}

	store := unitofwork.NewMemoryStore()
	ruleEngine := rules.NewEngine()
	mem := memory.NewStore()
	agentReg := agents.NewRegistry()

	agentReg.Register(agents.AgentVersion{Name: "steward", Role: "steward", Version: "0.1.0", ModelTier: "frontier"})
	agentReg.Register(agents.AgentVersion{Name: "worker-1", Role: "worker", Version: "0.1.0", ModelTier: "cheap"})
	agentReg.Register(agents.AgentVersion{Name: "judge", Role: "judge", Version: "0.1.0", ModelTier: "frontier"})

	for _, r := range domain.CatalogPack().DefaultRules {
		ruleEngine.Propose(rules.VersionedRule{Description: r, Source: "domain-pack", CreatedBy: "system"})
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

	steward := agents.NewSmartSteward(store)
	worker := agents.NewLLMWorker("worker-1", "catalog-corrector", gw, g, ruleEngine.ToAgentsRules())
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

		proposal, err := worker.Execute(uow)
		if err != nil {
			fmt.Printf("  Worker error: %v\n", err)
			continue
		}

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
