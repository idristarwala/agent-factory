package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/learning"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/queue"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/swarm"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func main() {
	g := graph.NewMemoryGraph()
	g.AddNode(graph.Node{ID: "item-1", Type: "item", Props: map[string]any{"status": "needs_fix", "title": "bad title"}})

	q := queue.NewMemoryQueue()
	re := rules.NewEngine()
	reg := agents.NewRegistry()
	ll := learning.NewLearningLoop(re, reg)

	store := unitofwork.NewMemoryStore()
	steward := agents.NewSmartSteward(store)

	// Real or noop gateway so CreateWorker succeeds
	var gw models.Gateway = &noopGateway{}
	if g2, err := models.NewOpenAICompatFromEnv(); err == nil {
		gw = g2
	}
	steward.SetRuntime(gw, g, nil)

	var judge agents.Judge = &demoJudge{}
	if _, ok := gw.(*noopGateway); !ok {
		// Prefer real LLM judge when a real gateway is available
		lmJudge := agents.NewLLMJudge(gw, g)
		// Use QuorumJudge with two independent evaluations when a real gateway is present
		judge = agents.NewQuorumJudge([]agents.Judge{lmJudge, lmJudge}, 2)
	}

	runner := swarm.NewContinuousRunner(steward, judge, q, g, re, ll)
	runner.Workers = 2
	runner.SetBudget(swarm.Budget{
		MaxRounds:    5,
		MaxWallClock: 10 * time.Minute,
		MaxFailures:  20,
	})

	goal := "Fix all entities until clean"
	if len(os.Args) > 1 {
		goal = os.Args[1]
	}

	fmt.Println("Continuous swarm with real gateway (stops when graph is clean)")
	if err := runner.Run(context.Background(), goal); err != nil {
		fmt.Println("Runner finished:", err)
	}
}

type demoJudge struct{}

func (j *demoJudge) Evaluate(m *graph.Mutation, rules []agents.Rule) (agents.Decision, error) {
	return agents.Decision{Approved: true, Reason: "demo approve"}, nil
}

type noopGateway struct{}

func (n *noopGateway) Generate(req models.Request) (*models.Response, error) {
	return &models.Response{Content: `{"title":"fixed title","status":"fixed"}`, Model: "noop"}, nil
}
