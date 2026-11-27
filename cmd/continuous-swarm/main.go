package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/tools"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/configuredgateway"
	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/learning"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/queue"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/settings"
	factoryStore "github.com/idristarwala/agent-factory/internal/store"
	"github.com/idristarwala/agent-factory/internal/swarm"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func main() {
	var fields []string
	flag.Func("field", "Editable record field (repeat for multiple fields)", func(name string) error {
		if name == "" || name == "status" || slices.Contains(fields, name) {
			return fmt.Errorf("expected a distinct editable field name")
		}
		fields = append(fields, name)
		return nil
	})
	flag.Parse()
	database, err := factoryStore.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	definitions := factoryStore.FactoryDefinitions(database, "")
	registry := agents.NewStoredRegistry(definitions)

	g := graph.NewMemoryGraph()
	g.AddNode(graph.Node{ID: "item-1", Type: "item", Props: map[string]any{"status": "needs_fix", "title": "bad title"}})

	q := queue.NewMemoryQueue()
	re := rules.NewStoredEngine(definitions)
	reg := registry
	ll := learning.NewLearningLoop(re, reg)

	store := unitofwork.NewMemoryStore()
	if _, err := registry.Resolve(agents.AgentVersion{Name: "steward", Role: "steward", ModelTier: "frontier", CreatedBy: "system"}); err != nil {
		log.Fatal(err)
	}
	steward := agents.NewSmartSteward(store)
	steward.Registry = registry
	steward.AllowedFields = fields

	// Real or noop gateway so CreateWorker succeeds
	var gw models.Gateway = &noopGateway{}
	realGateway := false
	for _, key := range []string{"LLM_GATEWAY_CONFIG", "LLM_PROVIDER", "LLM_BASE_URL", "LLM_API_KEY", "GEMINI_API_KEY", "LLM_JUDGE_PROVIDER", "LLM_JUDGE_MODEL"} {
		if os.Getenv(key) != "" {
			realGateway = true
			break
		}
	}
	if realGateway {
		gw, err = configuredgateway.NewFromEnv()
		if err != nil {
			log.Fatal(err)
		}
	}
	prices, err := factory.ParsePrices(os.Getenv("FACTORY_MODEL_PRICES"))
	if err != nil {
		log.Fatal(err)
	}
	meter := &factory.ContinuousUsage{Gateway: gw, Prices: prices}
	steward.SetRuntime(meter, g, nil)

	judgeDefinition, err := registry.Resolve(agents.AgentVersion{Name: "judge", Role: "judge", ModelTier: "frontier", CreatedBy: "system"})
	if err != nil {
		log.Fatal(err)
	}
	cfg, version, err := settings.Current(definitions)
	if err != nil {
		log.Fatal(err)
	}
	if version == 0 {
		cfg.MaxRounds = 5
		cfg.MaxMinutes = 10
	}
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	var judge agents.Judge = &demoJudge{}
	if realGateway {
		// Prefer real LLM judge when a real gateway is available
		lmJudge := agents.NewLLMJudge(meter, g)
		lmJudge.Definition = judgeDefinition
		evaluations := make([]agents.Judge, cfg.JudgeEvaluations)
		for i := range evaluations {
			evaluations[i] = lmJudge
		}
		judge = agents.NewQuorumJudge(evaluations, cfg.JudgeApprovals)
	}

	runner := swarm.NewContinuousRunner(steward, judge, q, g, re, ll)
	if err := configureEvidence(runner, os.Getenv("FACTORY_EVIDENCE_CONFIG"), os.Getenv("FACTORY_PACK")); err != nil {
		log.Fatal(err)
	}
	runner.Usage = meter
	runner.Workers = cfg.WorkerCount
	runner.Stagnation = cfg.StagnantRounds
	ll.ReplayGate = cfg.ReplayGate
	runner.SetObserve(cfg.Observe)
	runner.SetBudget(swarm.Budget{MaxRounds: cfg.MaxRounds, MaxWallClock: time.Duration(cfg.MaxMinutes) * time.Minute, MaxTokens: cfg.MaxTokens, MaxFailures: cfg.MaxFailures})
	fmt.Printf("Settings revision %d\n", version)

	goal := "Fix all entities until clean"
	if flag.NArg() > 0 {
		goal = strings.Join(flag.Args(), " ")
	}

	fmt.Println("Continuous swarm (stops when graph is clean)")
	started := time.Now()
	runErr := runner.Run(context.Background(), goal)
	ws := factory.Workspace{Store: definitions}
	run, err := ws.FinishContinuous(goal, started, cfg, version, meter, runner.Audit.All(), runErr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Run %s: %s (%d tokens)\n", run.ID, run.Status, meter.Tokens())
	if runErr != nil {
		fmt.Println("Runner finished:", runErr)
	}
}

func configureEvidence(runner *swarm.ContinuousRunner, path, pack string) error {
	pack = strings.TrimSpace(pack)
	if path == "" && pack == "" {
		return nil
	}
	if pack == "" || path == "" {
		return fmt.Errorf("set both FACTORY_EVIDENCE_CONFIG and FACTORY_PACK to use continuous runner grants")
	}
	cfg, err := tools.LoadEvidenceConfig(path, pack)
	if err != nil {
		return err
	}
	if !tools.CheckGrant(cfg.Grants, pack, "propose", "read-only") {
		return fmt.Errorf("pack %q has no read-only propose grant", pack)
	}
	runner.SetTools(cfg.Tools, cfg.Recipe)
	runner.SetGrants(pack, cfg.Grants)
	return nil
}

type demoJudge struct{}

func (j *demoJudge) Evaluate(m *graph.Mutation, rules []agents.Rule) (agents.Decision, error) {
	return agents.Decision{Approved: true, Reason: "demo approve"}, nil
}

type noopGateway struct{}

func (n *noopGateway) Generate(req models.Request) (*models.Response, error) {
	return &models.Response{Content: `{"title":"fixed title","status":"fixed"}`, Model: "noop"}, nil
}
