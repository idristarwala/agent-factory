package swarm

import (
	"context"
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

// ContinuousRunner runs the factory until problems are resolved or max rounds hit.
type ContinuousRunner struct {
	Steward   agents.Steward
	Judge     agents.Judge
	Queue     queue.WorkQueue
	Graph     graph.Graph
	Rules     *rules.Engine
	Learning  *learning.LearningLoop
	Workers   int
	MaxRounds int
	Observe   bool
	Grants    []tools.Grant
	Pack      string
	Tools     map[string]*tools.HTTPTool
	Recipe    tools.ContextRecipe
}

func NewContinuousRunner(s agents.Steward, j agents.Judge, q queue.WorkQueue, g graph.Graph, re *rules.Engine, ll *learning.LearningLoop) *ContinuousRunner {
	return &ContinuousRunner{
		Steward:   s,
		Judge:     j,
		Queue:     q,
		Graph:     g,
		Rules:     re,
		Learning:  ll,
		Workers:   4,
		MaxRounds: 10,
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

func (r *ContinuousRunner) Run(ctx context.Context, goal string) error {
	for round := 1; round <= r.MaxRounds; round++ {
		fmt.Printf("===== Round %d =====\n", round)

		uows, err := r.Steward.InterpretGoal(goal, r.Graph.Snapshot())
		if err != nil {
			return err
		}
		if len(uows) == 0 {
			fmt.Println("No more units of work. Done.")
			return nil
		}

		for _, uow := range uows {
			if err := r.Queue.Enqueue(ctx, uow); err != nil {
				return err
			}
		}

		var wg sync.WaitGroup
		decisions := make([]agents.Decision, 0)
		var mu sync.Mutex

		for i := 0; i < r.Workers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for {
					uow, err := r.Queue.Claim(ctx, fmt.Sprintf("worker-%d", workerID), 30*time.Second)
					if err != nil || uow == nil {
						return
					}

					// Grants check
					if len(r.Grants) > 0 && r.Pack != "" {
						if !tools.CheckGrant(r.Grants, r.Pack, "propose", "read-only") {
							r.Queue.Fail(ctx, uow.ID, "grant denied")
							continue
						}
					}

					// Gather context if recipe is set
					var evidence []string
					if r.Tools != nil && (r.Recipe.Entity != "" || r.Recipe.Neighbours != "") {
						args := map[string]string{"id": string(uow.InputSubgraph)}
						ctxResult, err := tools.ExecuteRecipe(ctx, r.Recipe, r.Tools, args)
						if err == nil && ctxResult != nil {
							evidence = append(evidence, "context gathered")
							fmt.Printf("context gathered\n")
						}
					}

					worker, err := r.Steward.CreateWorker(agents.WorkerSpec{Name: "generic", Role: "worker", ModelTier: "cheap"})
					if err != nil {
						r.Queue.Fail(ctx, uow.ID, "create worker: "+err.Error())
						continue
					}
					mutation, err := worker.Execute(uow)
					if err != nil {
						r.Queue.Fail(ctx, uow.ID, err.Error())
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
						r.Graph.Apply(mutation)
					}
					r.Queue.Complete(ctx, uow.ID)
					mu.Lock()
					decisions = append(decisions, decision)
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()

		if r.Learning != nil && len(decisions) > 0 {
			gaps := r.Learning.Review(decisions, nil)
			updates := r.Learning.ProposeUpdates(gaps)
			applied := r.Learning.Apply(updates)
			for _, a := range applied {
				fmt.Printf("Learning applied: %s\n", a)
			}
		}

		pending, _ := r.Queue.PendingCount(ctx)
		if pending == 0 {
			fmt.Println("Queue empty.")
		}
	}
	return fmt.Errorf("max rounds reached")
}
