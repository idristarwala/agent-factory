package main

import (
	"fmt"
	"os"
	"strings"

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
	// Heuristic: short/generic titles or descriptions, or empty material when expected
	if len(title) < 15 || strings.Contains(strings.ToLower(title), "shirt") && !strings.Contains(strings.ToLower(title), "nike") && !strings.Contains(strings.ToLower(title), "adidas") {
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

	for _, r := range domain.CatalogPack().DefaultRules {
		ruleEngine.Add(agents.Rule{Description: r, Source: "domain-pack"})
	}
	ruleEngine.Add(agents.Rule{
		Description: "Titles should include brand and key attributes (color, type). Descriptions should be specific.",
		Source:      "human",
	})
	ruleEngine.Add(agents.Rule{
		Description: "Do not include attributes that do not apply to the product type (e.g. battery_hours on footwear).",
		Source:      "human",
	})

	steward := agents.NewSmartSteward(store)
	worker := agents.NewLLMWorker("swarm-worker", "catalog-corrector", gw, g)
	judge := agents.NewLLMJudge(gw)

	fmt.Println("=== Catalog Swarm Correction ===")
	fmt.Println("Variant groups:")
	for group, ids := range catalog.VariantGroups() {
		fmt.Printf("  %s: %v\n", group, ids)
	}
	fmt.Println()

	nodes := catalog.Seed()
	corrected := 0
	for _, n := range nodes {
		if !needsWork(n) {
			fmt.Printf("SKIP (looks ok): %s — %v\n", n.ID, n.Props["title"])
			continue
		}

		fmt.Printf("\n--- Correcting %s (PT=%v Brand=%v) ---\n", n.ID, n.Props["pt"], n.Props["brand"])
		fmt.Printf("  Before: title=%q desc=%q\n", n.Props["title"], n.Props["description"])

		uows, err := steward.InterpretGoal(fmt.Sprintf("Fix title and description for %s", n.ID), nil)
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

		decision, err := judge.Evaluate(proposal, ruleEngine.All())
		if err != nil {
			fmt.Printf("  Judge error: %v\n", err)
			continue
		}

		if decision.Approved && len(proposal.Updates) > 0 {
			g.ProposeMutation(proposal)
			updated, _ := g.GetNode(n.ID)
			fmt.Printf("  After:  title=%q\n          desc=%q\n", updated.Props["title"], updated.Props["description"])
			fmt.Printf("  Judge: %s\n", decision.Reason)
			mem.Add("learning", fmt.Sprintf("%s corrected to: %v", n.ID, updated.Props["title"]), "swarm", nil)
			corrected++
		} else {
			fmt.Printf("  Rejected: %s\n", decision.Reason)
		}
	}

	fmt.Printf("\n=== Summary ===\nCorrected: %d\nLearnings: %d\nRules: %d\n", corrected, len(mem.ByKind("learning")), len(ruleEngine.All()))
}
