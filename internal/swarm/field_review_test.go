package swarm

// End-to-end tests for the continuous runner + learning loop.
// They use a scripted fake gateway: no network, no keys, no data files.

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/learning"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/queue"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// fakeGateway answers worker calls (requests that carry a Schema) with worker(),
// and judge calls with judge(). It records every worker request.
type fieldReviewGateway struct {
	mu       sync.Mutex
	calls    int
	worker   func(n int, req models.Request) string
	judge    func(req models.Request) string
	requests []models.Request
}

func (f *fieldReviewGateway) Generate(req models.Request) (*models.Response, error) {
	f.mu.Lock()
	f.calls++
	n := f.calls
	f.mu.Unlock()
	if strings.HasPrefix(req.System, "You are a strict judge") {
		return &models.Response{Content: f.judge(req), Model: "fake"}, nil
	}
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()
	return &models.Response{Content: f.worker(n, req), Model: "fake"}, nil
}

func fieldReviewRunner(t *testing.T, items int, gw *fieldReviewGateway, rounds int) (*ContinuousRunner, *rules.Engine, *learning.LearningLoop) {
	t.Helper()
	g := graph.NewMemoryGraph()
	for i := 0; i < items; i++ {
		g.AddNode(graph.Node{ID: graph.NodeID(fmt.Sprintf("item-%d", i)), Type: "product",
			Props: map[string]any{"status": "needs_fix", "title": "Red Shirt Sz M", "clothing_size": "M", "actual_color": "Red"}})
	}
	re := rules.NewEngine()
	steward := agents.NewSmartSteward(unitofwork.NewMemoryStore())
	steward.SetRuntime(gw, g, nil)
	ll := learning.NewLearningLoop(re, agents.NewRegistry())
	r := NewContinuousRunner(steward, agents.NewLLMJudge(gw, g), queue.NewMemoryQueue(), g, re, ll)
	r.Workers = 1
	r.SetBudget(Budget{MaxRounds: rounds, MaxWallClock: time.Minute, MaxFailures: 1000})
	return r, re, ll
}

func fieldReviewRules(re *rules.Engine) []rules.VersionedRule {
	var out []rules.VersionedRule
	for _, r := range re.Active() {
		if r.CreatedBy == "steward" {
			out = append(out, r)
		}
	}
	return out
}

// 1. With ReplayGate on, a learned rule must not go live before a replay shows it helps.
func TestFieldReviewerReplayGateHoldsLearnedRules(t *testing.T) {
	gw := &fieldReviewGateway{
		worker: func(int, models.Request) string { return `{"items":[{"id":1,"why":"x"}]}` }, // wrong shape
		judge:  func(models.Request) string { return "APPROVE: ok" },
	}
	r, re, ll := fieldReviewRunner(t, 5, gw, 1)
	ll.ReplayGate = true
	_ = r.Run(context.Background(), "Fix product titles")
	if len(gw.requests) == 0 || len(re.History("")) != 1 {
		t.Fatal("expected one pending rule after worker failures")
	}
	if got := fieldReviewRules(re); len(got) != 0 {
		t.Fatalf("ReplayGate=true but %d learned rule(s) are active, e.g. %q", len(got), got[0].Description)
	}
}

// 2. Many failures with the same cause in one round should produce at most one rule.
// The worker error includes the reply length, so replies of different length must still group.
func TestFieldReviewerSameCauseFailuresMakeOneRule(t *testing.T) {
	gw := &fieldReviewGateway{
		worker: func(n int, _ models.Request) string {
			return fmt.Sprintf(`{"items":[{"id":%d,"why":"%s"}]}`, n, strings.Repeat("x", n)) // wrong shape, varying length
		},
		judge: func(models.Request) string { return "APPROVE: ok" },
	}
	r, re, _ := fieldReviewRunner(t, 10, gw, 1)
	_ = r.Run(context.Background(), "Fix product titles")
	if got := fieldReviewRules(re); len(got) != 1 {
		t.Fatalf("10 format failures with one cause produced %d rules; want exactly one", len(got))
	}
}

// 3. An item the judge rejects once in every round must eventually teach the loop something.
func TestFieldReviewerRejectionsAcrossRoundsAreLearned(t *testing.T) {
	gw := &fieldReviewGateway{
		worker: func(int, models.Request) string { return `{"title":"Red Shirt Size XL"}` },
		judge:  func(models.Request) string { return "REJECT: title states size XL but the item's size is M" },
	}
	r, re, _ := fieldReviewRunner(t, 1, gw, 3)
	_ = r.Run(context.Background(), "Fix product titles")
	if got := fieldReviewRules(re); len(got) == 0 {
		t.Fatalf("item rejected in every round for the same reason, but nothing was learned")
	}
}

// 4. A title task should only expose title/description in the reply schema,
// otherwise the model fills generic attributes (size, color) on every item.
func TestFieldReviewerTitleTaskSchemaOnlyAllowsTitleFields(t *testing.T) {
	gw := &fieldReviewGateway{
		worker: func(int, models.Request) string { return `{"title":"Red Shirt, Size M"}` },
		judge:  func(models.Request) string { return "APPROVE: ok" },
	}
	r, _, _ := fieldReviewRunner(t, 1, gw, 1)
	_ = r.Run(context.Background(), "Fix product titles")
	if len(gw.requests) == 0 || gw.requests[0].Schema == nil {
		t.Fatalf("worker sent no schema")
	}
	props, _ := gw.requests[0].Schema["properties"].(map[string]any)
	for k := range props {
		if k != "title" && k != "description" {
			t.Errorf("title task schema allows %q", k)
		}
	}
}

// 5. Learned rules must reach the worker's prompt in the next round.
func TestFieldReviewerLearnedRuleReachesWorkerNextRound(t *testing.T) {
	gw := &fieldReviewGateway{
		worker: func(int, models.Request) string { return `{"items":[]}` }, // always wrong shape
		judge:  func(models.Request) string { return "APPROVE: ok" },
	}
	r, re, _ := fieldReviewRunner(t, 3, gw, 2)
	_ = r.Run(context.Background(), "Fix product titles")
	learned := fieldReviewRules(re)
	if len(learned) == 0 {
		t.Fatal("nothing learned in round 1; see TestSameCauseFailuresMakeOneRule")
	}
	last := gw.requests[len(gw.requests)-1].User
	if !strings.Contains(last, learned[0].Description) {
		t.Fatalf("learned rule %q not in the worker prompt of a later round", learned[0].Description)
	}
}
