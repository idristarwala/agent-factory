package swarm

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/learning"
	"github.com/idristarwala/agent-factory/internal/queue"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/tools"
)

// Budget caps the continuous run.
type Budget struct {
	MaxRounds    int
	MaxWallClock time.Duration
	MaxFailures  int
	MaxTokens    int
}

// ContinuousRunner runs until the goal is achieved or a limit is hit.
type ContinuousRunner struct {
	Steward    agents.Steward
	Judge      agents.Judge
	Queue      queue.WorkQueue
	Graph      graph.Graph
	Rules      *rules.Engine
	Learning   *learning.LearningLoop
	Workers    int
	Budget     Budget
	Observe    bool
	Grants     []tools.Grant
	Pack       string
	Tools      map[string]*tools.HTTPTool
	Recipe     tools.ContextRecipe
	Stagnation int
	progress   *Progress
}

func NewContinuousRunner(s agents.Steward, j agents.Judge, q queue.WorkQueue, g graph.Graph, re *rules.Engine, ll *learning.LearningLoop) *ContinuousRunner {
	if smart, ok := s.(*agents.SmartSteward); ok && ll != nil && ll.Registry != nil {
		smart.Registry = ll.Registry
	}
	return &ContinuousRunner{
		Steward:  s,
		Judge:    j,
		Queue:    q,
		Graph:    g,
		Rules:    re,
		Learning: ll,
		Workers:  4,
		Budget: Budget{
			MaxRounds:    10,
			MaxWallClock: 30 * time.Minute,
			MaxFailures:  100,
			MaxTokens:    1000000,
		},
		Stagnation: 2,
		progress:   NewProgress(),
	}
}

func (r *ContinuousRunner) SetObserve(enabled bool) { r.Observe = enabled }

func (r *ContinuousRunner) SetGrants(pack string, grants []tools.Grant) {
	r.Pack = pack
	r.Grants = grants
}

func (r *ContinuousRunner) SetTools(reg map[string]*tools.HTTPTool, recipe tools.ContextRecipe) {
	r.Tools = reg
	r.Recipe = recipe
}

func (r *ContinuousRunner) SetBudget(b Budget) {
	r.Budget = b
}

func (r *ContinuousRunner) Run(ctx context.Context, goal string) error {
	if r.progress == nil {
		r.progress = NewProgress()
	}
	r.progress.MaxStagnant = r.Stagnation
	budgets := NewBudgets(r.Budget.MaxRounds, r.Budget.MaxWallClock, r.Budget.MaxTokens)
	totalFailures := 0

	for round := 1; round <= r.Budget.MaxRounds; round++ {
		if reason := budgets.Exhausted(round); reason != "" {
			return fmt.Errorf("budget exceeded: %s", reason)
		}
		if r.Budget.MaxFailures > 0 && totalFailures >= r.Budget.MaxFailures {
			return fmt.Errorf("failure budget exceeded (%d)", totalFailures)
		}

		fmt.Printf("===== Round %d =====\n", round)

		if smart, ok := r.Steward.(*agents.SmartSteward); ok && r.Rules != nil {
			active := r.Rules.ToAgentsRules()
			smart.SetRuntime(smart.Gateway, smart.Graph, active)
		}

		uows, err := r.Steward.InterpretGoal(goal, r.Graph.Snapshot())
		if err != nil {
			return err
		}
		if len(uows) == 0 {
			if r.verifyClean(goal) {
				fmt.Println("Goal achieved. Graph verified clean.")
				return nil
			}
			fmt.Println("Steward reports clean but verifier disagrees. Continuing.")
		}

		blocked := map[string]bool{}
		for _, id := range r.progress.BlockedNodes() {
			blocked[string(id)] = true
		}

		for _, uow := range uows {
			if blocked[string(uow.InputSubgraph)] {
				continue
			}
			if err := r.Queue.Enqueue(ctx, uow); err != nil {
				return err
			}
		}

		var wg sync.WaitGroup
		workerErrors := make([]error, r.Workers)
		decisions := make([]agents.Decision, 0)
		var failureMsgs []string
		var failureErrs []error
		var mu sync.Mutex

		for i := 0; i < r.Workers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for {
					uow, err := r.Queue.Claim(ctx, fmt.Sprintf("worker-%d", workerID), 30*time.Second)
					if err != nil {
						workerErrors[workerID] = fmt.Errorf("claim work: %w", err)
						return
					}
					if uow == nil {
						return
					}

					if len(r.Grants) > 0 && r.Pack != "" {
						if !tools.CheckGrant(r.Grants, r.Pack, "propose", "read-only") {
							r.Queue.Fail(ctx, uow.ID, "grant denied")
							continue
						}
					}

					var evidence []string
					if r.Tools != nil && (r.Recipe.Entity != "" || r.Recipe.Neighbours != "") {
						args := map[string]string{"id": string(uow.InputSubgraph)}
						ctxResult, err := tools.ExecuteRecipe(ctx, r.Recipe, r.Tools, args)
						if err == nil && ctxResult != nil {
							if entity, ok := ctxResult["entity"].(string); ok && entity != "" {
								evidence = append(evidence, "entity: "+entity[:min(200, len(entity))])
							}
							if neigh, ok := ctxResult["neighbours"].(string); ok && neigh != "" {
								evidence = append(evidence, "neighbours: "+neigh[:min(200, len(neigh))])
							}
						}
					}

					if len(evidence) > 0 {
						uow.Evidence = evidence
					}

					worker, err := r.Steward.CreateWorker(agents.WorkerSpec{Name: "generic", Role: "worker", ModelTier: "cheap"})
					if err != nil {
						r.Queue.Fail(ctx, uow.ID, "create worker: "+err.Error())
						mu.Lock()
						failureMsgs = append(failureMsgs, err.Error())
						failureErrs = append(failureErrs, err)
						mu.Unlock()
						continue
					}
					mutation, err := worker.Execute(uow)
					if err != nil {
						r.Queue.Fail(ctx, uow.ID, err.Error())
						mu.Lock()
						failureMsgs = append(failureMsgs, err.Error())
						failureErrs = append(failureErrs, err)
						if r.progress.RecordFailure(string(uow.InputSubgraph)) {
							fmt.Printf("Unit %s blocked after repeated failures\n", uow.InputSubgraph)
						}
						mu.Unlock()
						continue
					}
					if mutation != nil {
						mutation.Evidence = append(mutation.Evidence, evidence...)
					}
					decision, err := r.Judge.Evaluate(mutation, r.Rules.ToAgentsRules())
					if err != nil {
						r.Queue.Fail(ctx, uow.ID, err.Error())
						continue
					}
					if decision.Approved && !r.Observe {
						if err := r.Graph.Apply(mutation); err != nil {
							writeErr := fmt.Errorf("apply mutation for unit %s: %w", uow.ID, err)
							workerErrors[workerID] = errors.Join(writeErr, r.Queue.Fail(ctx, uow.ID, writeErr.Error()))
							return
						}
					}
					if err := r.Queue.Complete(ctx, uow.ID); err != nil {
						workerErrors[workerID] = fmt.Errorf("complete unit %s: %w", uow.ID, err)
						return
					}
					mu.Lock()
					decisions = append(decisions, decision)
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()
		if err := errors.Join(workerErrors...); err != nil {
			return err
		}
		// Retain the existing per-decision estimate until gateways report actual usage.
		budgets.AddTokens(800 * len(decisions))

		totalFailures += len(failureMsgs)

		if r.Learning != nil {
			signals := CollectFailureSignals(nil, failureErrs)
			_ = signals
			gaps := r.Learning.Review(decisions, failureMsgs)
			updates := r.Learning.ProposeUpdates(gaps)
			applied := r.Learning.Apply(updates)
			for _, a := range applied {
				fmt.Printf("Learning applied: %s\n", a)
			}
			if len(failureMsgs) > 0 {
				fmt.Printf("Worker failures this round: %d (total %d)\n", len(failureMsgs), totalFailures)
			}
		}

		if r.progress.Observe(r.Graph.Snapshot()) {
			fmt.Printf("No progress for %d rounds. Stopping.\n", r.progress.Stagnant)
			return fmt.Errorf("stagnation: no progress for %d rounds", r.progress.Stagnant)
		}

		pending, _ := r.Queue.PendingCount(ctx)
		if pending == 0 {
			fmt.Println("Queue empty.")
		}
	}
	return fmt.Errorf("max rounds reached")
}

func (r *ContinuousRunner) verifyClean(goal string) bool {
	uows, err := r.Steward.InterpretGoal(goal, r.Graph.Snapshot())
	return err == nil && len(uows) == 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
