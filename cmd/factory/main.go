package main

import (
	"fmt"
	"os"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/domain"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/store"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func main() {
	mode := store.Mode(os.Getenv("STORE_MODE"))
	if mode == "" {
		mode = store.ModeMemory
	}
	dsn := os.Getenv("STORE_DSN")

	st, err := store.Open(mode, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	fmt.Printf("Store mode: %s\n", mode)

	gw, gwErr := models.NewGatewayFromEnv()
	if gwErr != nil {
		fmt.Printf("gateway: %v (continuing without)\n", gwErr)
		gw = nil
	}

	ruleEngine := rules.NewEngine()
	agentReg := agents.NewRegistry()

	// Register agents with versions
	agentReg.Register(agents.AgentVersion{Name: "steward-1", Role: "steward", Version: "0.1.0", ModelTier: "frontier"})
	agentReg.Register(agents.AgentVersion{Name: "judge-1", Role: "judge", Version: "0.1.0", ModelTier: "frontier"})
	agentReg.Register(agents.AgentVersion{Name: "worker-1", Role: "worker", Version: "0.1.0", ModelTier: "cheap"})

	// Load rules from all domain packs (domain-agnostic)
	for _, pack := range domain.All() {
		for _, r := range pack.DefaultRules {
			ruleEngine.Propose(rules.VersionedRule{
				Description: r,
				Source:      "domain-pack:" + pack.Name,
				CreatedBy:   "system",
			})
		}
	}

	fmt.Println(agentReg.Summary())
	fmt.Println(ruleEngine.Summary())

	// Example: create a unit of work and a node (any domain)
	if len(os.Args) > 1 {
		goal := os.Args[1]
		u := &unitofwork.UnitOfWork{
			ID:          "uow-1",
			Description: goal,
			Status:      unitofwork.StatusProposed,
		}
		if err := st.CreateUnit(u); err != nil {
			fmt.Fprintf(os.Stderr, "create unit: %v\n", err)
		} else {
			fmt.Printf("Created unit of work for goal: %s\n", goal)
		}

		// Seed a generic node
		st.UpsertNode(graph.Node{ID: "item-1", Type: "entity", Props: map[string]any{"name": "example"}})
		fmt.Println("Seeded generic node item-1")
	}

	_ = gw // gateway available for workers
	fmt.Println("Factory ready (domain-agnostic).")
}
