package main

import (
	"context"
	"fmt"
	"os"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/learning"
	"github.com/idristarwala/agent-factory/internal/queue"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/swarm"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func main() {
	g := graph.NewMemoryGraph()
	// Seed a node that needs fixing
	g.Upsert(graph.Node{ID: "item-1", Type: "item", Props: map[string]any{"status": "needs_fix", "title": "bad title"}})

	q := queue.NewMemoryQueue()
	re := rules.NewEngine()
	reg := agents.NewRegistry()
	ll := learning.NewLearningLoop(re, reg)

	store := unitofwork.NewMemoryStore()
	steward := agents.NewSmartSteward(store)
	steward.Graph = g // enables graph-clean detection

	judge := &demoJudge{}

	runner := swarm.NewContinuousRunner(steward, judge, q, g, re, ll)
	runner.Workers = 2
	runner.MaxRounds = 5

	goal := "Fix all entities until clean"
	if len(os.Args) > 1 {
		goal = os.Args[1]
	}

	fmt.Println("Continuous swarm with SmartSteward (stops when graph is clean)")
	if err := runner.Run(context.Background(), goal); err != nil {
		fmt.Println("Runner finished:", err)
	}
	fmt.Println("Final graph snapshot:")
	for _, n := range g.Snapshot() {
		fmt.Printf("  %s: %v\n", n.ID, n.Props)
	}
}

type demoJudge struct{}

func (j *demoJudge) Evaluate(m *graph.Mutation, rules []agents.Rule) (agents.Decision, error) {
	return agents.Decision{Approved: true, Reason: "demo approve"}, nil
}
