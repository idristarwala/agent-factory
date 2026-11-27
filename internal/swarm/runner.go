package swarm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
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
	MaxFailures  int // stop if total failures exceed this
}

// ContinuousRunner runs until the goal is achieved or a limit is hit.
type ContinuousRunner struct {
	Steward     agents.Steward
	Judge       agents.Judge
	Queue       queue.WorkQueue
	Graph       graph.Graph
	Rules       *rules.Engine
	Learning    *learning.LearningLoop
	Workers     int
	Budget      Budget
	Observe     bool
	Grants      []tools.Grant
	Pack        string
	Tools       map[string]*tools.HTTPTool
	Recipe      tools.ContextRecipe
	Stagnation  int // stop after this many unchanged rounds (default 2)
}

func NewContinuousRunner(s agents.Steward, j agents.Judge, q queue.WorkQueue, g graph.Graph, re *rules.Engine, ll *learning.LearningLoop) *ContinuousRunner {
	return &ContinuousRunner{
		Steward: s,
		Judge:   j,
		Queue:   q,
		Graph:   g,
		Rules:   re,
		Learning: ll,
		Workers: 4,
		Budget: Budget{
			MaxRounds:    10,
			MaxWallClock: 30 * time.Minute,
			MaxFailures:  100,
		},
		Stagnation: 2,
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

func (r *ContinuousRunner) SetBudget(b Budget) { r.Budget = b }

func (r *ContinuousRunner) Run(ctx context.Context, goal string) error {
	start := time.Now()
	var lastHash string
	stagnationCount := 0
	totalFailures := 0

	for round := 1; round <= r.Budget.MaxRounds; round++ {
		// Budget checks
		if r.Budget.MaxWallClock > 0 && time.Since(start) > r.Budget.MaxWallClock {
			return fmt.Errorf("wall-clock budget exceeded")
		}
		if r.Budget.MaxFailures > 0 && totalFailures >= r.Budget.MaxFailures {
			return fmt.Errorf("failure budget exceeded (%d)", totalFailures)
		}

		fmt.Printf("===== Round %d =====\n", round)

		// Refresh active rules
		if smart, ok := r.Steward.(*agents.SmartSteward); ok && r.Rules != nil {
			var active []agents.Rule
			for _, vr := range r.Rules.Active() {
				active = append(active, agents.Rule{Description: vr.Description})
			}
			smart.SetRuntime(smart.Gateway, smart.Graph, active)
		}

		uows, err := r.Steward.InterpretGoal(goal, r.Graph.Snapshot())
		if err != nil {
			return err
		}
		if len(uows) == 0 {
			// Verifier gate: confirm the graph is truly clean
			if r.verifyClean(goal) {
				fmt.Println("Goal achieved. Graph verified clean.")
				return nil
			}
			fmt.Println("Steward reports clean but verifier disagrees. Continuing.")
		}

		for _, uow := range uows {
			if err := r.Queue.Enqueue(ctx, uow); err != nil {
				return err
			}
		}

		var wg sync.WaitGroup
		decisions := make([]agents.Decision, 0)
		var failureMsgs []string
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
						mu.Unlock()
						continue
					}
					mutation, err := worker.Execute(uow)
					if err != nil {
						r.Queue.Fail(ctx, uow.ID, err.Error())
						mu.Lock()
						failureMsgs = append(failureMsgs, err.Error())
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

		totalFailures += len(failureMsgs)

		// Learning
		if r.Learning != nil {
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

		// Progress delta
		currentHash := r.graphHash()
		if currentHash == lastHash {
			stagnationCount++
			fmt.Printf("No progress (stagnation %d/%d)\n", stagnationCount, r.Stagnation)
			if stagnationCount >= r.Stagnation {
				return fmt.Errorf("stagnation: no progress for %d rounds", stagnationCount)
			}
		} else {
			stagnationCount = 0
		}
		lastHash = currentHash

		pending, _ := r.Queue.PendingCount(ctx)
		if pending == 0 {
			fmt.Println("Queue empty.")
		}
	}
	return fmt.Errorf("max rounds reached")
}

// graphHash returns a stable hash of the current graph state.
func (r *ContinuousRunner) graphHash() string {
	snap := r.Graph.Snapshot()
	b, _ := json.Marshal(snap)
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h[:8])
}

// verifyClean is the verifier gate. Returns true only if the Steward still sees no work.
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
