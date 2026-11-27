package swarm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/obs"
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
	MaxRounds         int
	MaxWallClock      time.Duration
	MaxFailures       int
	MaxRecordFailures int
	MaxTokens         int
}

// ContinuousRunner runs until the goal is achieved or a limit is hit.
type ContinuousRunner struct {
	Audit      *obs.AuditLog
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
	Usage      interface {
		Tokens() int
		RecordTool(string)
	}
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
	r.Audit = obs.NewAuditLog()
	progress := NewProgress()
	if r.Budget.MaxRecordFailures > 0 {
		progress.MaxUnitFails = r.Budget.MaxRecordFailures
	}
	progress.MaxStagnant = r.Stagnation
	budgets := NewBudgets(r.Budget.MaxRounds, r.Budget.MaxWallClock, r.Budget.MaxTokens)
	initialTokens := 0
	if r.Usage != nil {
		initialTokens = r.Usage.Tokens()
	}
	totalFailures := 0
	previousPatterns := map[graph.NodeID]string{}

	for round := 1; round <= r.Budget.MaxRounds; round++ {
		if reason := budgets.Exhausted(round); reason != "" {
			return fmt.Errorf("budget exceeded: %s", reason)
		}
		if r.Budget.MaxFailures > 0 && totalFailures >= r.Budget.MaxFailures {
			return fmt.Errorf("failure budget exceeded (%d)", totalFailures)
		}

		fmt.Printf("===== Round %d =====\n", round)

		var active []agents.Rule
		if r.Rules != nil {
			active = r.Rules.ToAgentsRules()
		}
		if smart, ok := r.Steward.(*agents.SmartSteward); ok && r.Rules != nil {
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
		for _, id := range progress.BlockedNodes() {
			blocked[string(id)] = true
		}

		blockedCount := 0
		for _, uow := range uows {
			if blocked[string(uow.InputSubgraph)] {
				blockedCount++
				continue
			}
			if err := r.Queue.Enqueue(ctx, uow); err != nil {
				return err
			}
		}

		if len(uows) > 0 && blockedCount == len(uows) {
			return fmt.Errorf("all %d remaining work units are blocked by the per-record failure limit", blockedCount)
		}
		var wg sync.WaitGroup
		workerErrors := make([]error, r.Workers)
		decisions := make([]agents.Decision, 0)
		var failureErrs []error
		nextPatterns := map[graph.NodeID]string{}
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

					audit := func(actor, action, reason string) {
						r.Audit.Append(obs.AuditEntry{TraceID: uow.ID, EntityID: string(uow.InputSubgraph), Actor: actor, Action: action, Reason: reason})
					}
					if len(r.Grants) > 0 && r.Pack != "" {
						if !tools.CheckGrant(r.Grants, r.Pack, "propose", "read-only") {
							audit("worker", "blocked", "grant denied")
							r.Queue.Fail(ctx, uow.ID, "grant denied")
							continue
						}
					}

					var evidence []string
					if r.Recipe.Entity != "" || r.Recipe.Neighbours != "" {
						var props map[string]any
						if n, err := r.Graph.GetNode(uow.InputSubgraph); err == nil {
							props = n.Props
						}
						args := tools.RecordParams(string(uow.InputSubgraph), props)
						result, err := r.GatherEvidence(ctx, r.Recipe, r.Tools, args)
						if err != nil {
							audit("worker", "blocked", "source evidence unavailable")
							workerErrors[workerID] = errors.Join(err, r.Queue.Fail(ctx, uow.ID, err.Error()))
							return
						}
						record, err := r.Recipe.Collect(ctx, result)
						if err != nil {
							audit("worker", "blocked", "source evidence unavailable")
							workerErrors[workerID] = errors.Join(err, r.Queue.Fail(ctx, uow.ID, err.Error()))
							return
						}
						evidence = record.Text
						uow.Images = append(uow.Images, record.Images...)
						uow.Evidence = append(uow.Evidence, evidence...)
					}

					spec := agents.WorkerSpec{Name: "generic", Role: "worker", ModelTier: "cheap"}
					if r.Learning != nil {
						spec, err = r.Learning.WorkerForPattern(previousPatterns[uow.InputSubgraph], active)
						if err != nil {
							workerErrors[workerID] = errors.Join(err, r.Queue.Fail(ctx, uow.ID, "worker lookup failed"))
							return
						}
					}
					worker, err := r.Steward.CreateWorker(spec)
					if err != nil {
						audit("worker", "failed", "worker creation failed")
						r.Queue.Fail(ctx, uow.ID, "create worker: "+err.Error())
						mu.Lock()
						failureErrs = append(failureErrs, err)
						mu.Unlock()
						continue
					}
					mutation, err := worker.Execute(uow)
					if errors.Is(err, models.ErrTokenBudget) {
						audit("runner", "budget_exhausted", err.Error())
						workerErrors[workerID] = errors.Join(err, r.Queue.Fail(ctx, uow.ID, err.Error()))
						return
					}
					if err == nil && mutation == nil {
						err = errors.New("worker returned an empty proposal")
					}
					if err != nil {
						audit("worker", "failed", "worker produced no valid proposal")
						r.Queue.Fail(ctx, uow.ID, err.Error())
						mu.Lock()
						failureErrs = append(failureErrs, err)
						nextPatterns[uow.InputSubgraph] = "worker failure: " + learning.FailureCause(err)
						if progress.RecordFailure(string(uow.InputSubgraph)) {
							audit("runner", "blocked", "per-record worker failure limit reached")
							fmt.Printf("Unit %s blocked after repeated failures\n", uow.InputSubgraph)
						}
						mu.Unlock()
						continue
					}
					mutation.Evidence = append(mutation.Evidence, evidence...)
					snapshot, err := json.Marshal(mutation)
					if err != nil {
						audit("worker", "failed", "proposal could not be recorded")
						workerErrors[workerID] = errors.Join(err, r.Queue.Fail(ctx, uow.ID, "invalid proposal"))
						return
					}
					r.Audit.Append(obs.AuditEntry{Actor: "worker", Action: "proposed", Reason: "submitted for judge review", TraceID: uow.ID, EntityID: string(uow.InputSubgraph), Proposal: snapshot})
					decision, err := r.Judge.Evaluate(mutation, active)
					if errors.Is(err, models.ErrTokenBudget) {
						audit("runner", "budget_exhausted", err.Error())
						workerErrors[workerID] = errors.Join(err, r.Queue.Fail(ctx, uow.ID, err.Error()))
						return
					}
					if err != nil {
						audit("judge", "failed", "judge evaluation failed")
						r.Queue.Fail(ctx, uow.ID, err.Error())
						continue
					}
					action := "rejected"
					if decision.Approved {
						action = "approved"
					}
					r.Audit.Append(obs.AuditEntry{TraceID: uow.ID, EntityID: string(uow.InputSubgraph), Actor: "judge", Action: action, Reason: decision.Reason, Feedback: decision.Feedback})
					if decision.Approved && !r.Observe {
						if err := r.Graph.Apply(mutation); err != nil {
							audit("runner", "apply_failed", "graph write failed")
							writeErr := fmt.Errorf("apply mutation for unit %s: %w", uow.ID, err)
							workerErrors[workerID] = errors.Join(writeErr, r.Queue.Fail(ctx, uow.ID, writeErr.Error()))
							return
						}
					}
					if decision.Approved {
						if r.Observe {
							audit("runner", "observed", "observe mode")
						} else {
							audit("runner", "applied", "graph write succeeded")
						}
					}
					if err := r.Queue.Complete(ctx, uow.ID); err != nil {
						workerErrors[workerID] = fmt.Errorf("complete unit %s: %w", uow.ID, err)
						return
					}
					mu.Lock()
					if !decision.Approved {
						nextPatterns[uow.InputSubgraph] = learning.RejectionPattern(decision.Reason)
					}
					decisions = append(decisions, decision)
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()
		previousPatterns = nextPatterns
		if err := errors.Join(workerErrors...); err != nil {
			return err
		}

		totalFailures += len(failureErrs)

		if r.Learning != nil {
			signals := CollectFailureSignals(nil, failureErrs)
			gaps := r.Learning.ReviewSignals(decisions, signals)
			updates := r.Learning.ProposeUpdates(gaps)
			applied, err := r.Learning.Apply(updates)
			if err != nil {
				return fmt.Errorf("save learned updates: %w", err)
			}
			for _, a := range applied {
				fmt.Printf("Learning applied: %s\n", a)
			}
			approved := 0
			for _, decision := range decisions {
				if decision.Approved {
					approved++
				}
			}
			fmt.Printf("Round %d: %d units, %d approved, %d rejected, %d worker failures\n", round, len(uows), approved, len(decisions)-approved, len(failureErrs))
			for _, signal := range signals {
				fmt.Printf("Worker failure cause %s: %d\n", signal.Cause, signal.Count)
			}
		}

		if r.Usage != nil {
			budgets.TokensUsed = r.Usage.Tokens() - initialTokens
		}

		snapshot := r.Graph.Snapshot()
		if verifier, ok := r.Steward.(interface{ GoalComplete(any) bool }); ok && verifier.GoalComplete(snapshot) {
			fmt.Println("Goal achieved. Graph verified clean.")
			return nil
		}
		if progress.Observe(snapshot) {
			fmt.Printf("No progress for %d rounds. Stopping.\n", progress.Stagnant)
			return fmt.Errorf("stagnation: no progress for %d rounds", progress.Stagnant)
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
