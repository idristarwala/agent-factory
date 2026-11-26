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
	q := queue.NewMemoryQueue()
	re := rules.NewEngine()
	reg := agents.NewRegistry()
	ll := learning.NewLearningLoop(re, reg)

	steward := &stubSteward{}
	judge := &stubJudge{}

	runner := swarm.NewContinuousRunner(steward, judge, q, g, re, ll)
	runner.Workers = 2
	runner.MaxRounds = 3

	goal := "Fix all entities until clean"
	if len(os.Args) > 1 {
		goal = os.Args[1]
	}

	if err := runner.Run(context.Background(), goal); err != nil {
		fmt.Println("Runner finished:", err)
	}
}

type stubSteward struct{}

func (s *stubSteward) InterpretGoal(goal string, snap any) ([]*unitofwork.UnitOfWork, error) {
	return []*unitofwork.UnitOfWork{{
		ID:          "demo-uow-1",
		Description: goal,
	}}, nil
}

func (s *stubSteward) CreateWorker(spec agents.WorkerSpec) (agents.Worker, error) {
	return &stubWorker{}, nil
}

func (s *stubSteward) SetAutonomyMode(mode string) error { return nil }

type stubWorker struct{}

func (w *stubWorker) Claim(u *unitofwork.UnitOfWork) error { return nil }

func (w *stubWorker) Execute(u *unitofwork.UnitOfWork) (*graph.Mutation, error) {
	return &graph.Mutation{
		UnitOfWorkID: u.ID,
		Updates: []graph.Node{{
			ID:   "demo-entity",
			Type: "entity",
			Props: map[string]any{"status": "fixed"},
		}},
	}, nil
}

type stubJudge struct{}

func (j *stubJudge) Evaluate(m *graph.Mutation, rules []agents.Rule) (agents.Decision, error) {
	return agents.Decision{Approved: true, Reason: "demo approve"}, nil
}
