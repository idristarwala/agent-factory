package factory

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/learning"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/queue"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/schema"
	"github.com/idristarwala/agent-factory/internal/swarm"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

type heldOutGateway struct{ regression bool }

func (g heldOutGateway) Generate(req models.Request) (*models.Response, error) {
	result := "APPROVE: verified"
	if req.Schema != nil {
		result = `{"team":"support"}`
		switch {
		case strings.HasPrefix(req.UnitID, "replay/"):
			learned := strings.Contains(req.User, "Return only the allowed fields as flat JSON")
			if (!learned && strings.HasSuffix(req.UnitID, "/held-a")) || (g.regression && learned && strings.HasSuffix(req.UnitID, "/held-b")) {
				result = `{"team":"other"}`
			}
		case !strings.Contains(req.User, "Return only the allowed fields as flat JSON"):
			result = `{"invalid_field":true}`
		}
	}
	return &models.Response{Content: result, Model: "fixture"}, nil
}

type approvedFixture struct{}

func (approvedFixture) Generate(req models.Request) (*models.Response, error) {
	result := "APPROVE: verified"
	if req.Schema != nil {
		result = `{"team":"support"}`
	}
	return &models.Response{Content: result}, nil
}

func TestRunnerPromotesOnlyAfterHeldOutGain(t *testing.T) {
	st := revision.NewMemory()
	_, err := revision.SaveDocument(st, "factory/desk", 0, "human", []byte(`{"record_type":"ticket","fields":{"team":"string"}}`))
	if err != nil {
		t.Fatal(err)
	}
	ws := Workspace{Store: st, Gateway: approvedFixture{}}
	var records []schema.Entity
	for _, id := range []string{"held-a", "held-b"} {
		records = append(records, schema.Entity{ID: id, Type: "ticket", Attributes: map[string]any{"team": "unknown"}})
	}
	run, err := ws.Start(RunRequest{Description: "Assign the support team", Factory: "desk", Records: records})
	if err != nil {
		t.Fatal(err)
	}
	if err = ws.Execute(run.ID); err != nil {
		t.Fatal(err)
	}
	run, err = ws.Get(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range run.Decisions {
		run, err = ws.Decide(run.ID, d.ID, "approved", "", run.Version)
		if err != nil {
			t.Fatal(err)
		}
	}
	ref := fmt.Sprintf("%s@%d", run.ID, run.Version)
	live := graph.NewMemoryGraph()
	live.AddNode(graph.Node{ID: "live", Type: "ticket", Props: map[string]any{"team": "unknown", "status": "open"}})
	gateway := heldOutGateway{}
	ws.Gateway = gateway
	replay, err := ws.ReviewedReplay(ref, live, nil)
	if err != nil {
		t.Fatal(err)
	}
	re := rules.NewStoredEngine(st)
	loop := learning.NewLearningLoop(re, agents.NewStoredRegistry(st))
	loop.ReplayGate, loop.Replay = true, replay
	steward := agents.NewSmartSteward(unitofwork.NewMemoryStore())
	steward.SetRuntime(gateway, live, nil)
	runner := swarm.NewContinuousRunner(steward, agents.NewLLMJudge(gateway, live), queue.NewMemoryQueue(), live, re, loop)
	runner.Workers = 1
	if err := runner.Run(t.Context(), "Assign the support team"); err != nil {
		t.Fatal(err)
	}
	active := re.Active()
	if len(active) != 1 {
		t.Fatal(active)
	}
	r := active[0]
	h := re.History(r.ID)
	if len(h) != 2 || h[0].Active || h[0].Replay != nil || r.Version != 2 || r.ActivatedFrom != 1 || r.Replay == nil || r.Replay.Run != run.ID || r.Replay.Revision != run.Version || !reflect.DeepEqual(r.Replay.Cases, []rules.ReplayCase{{RecordID: "held-a", CandidatePassed: true}, {RecordID: "held-b", BaselinePassed: true, CandidatePassed: true}}) {
		t.Fatal(h, r.Replay)
	}
	node, err := live.GetNode("live")
	if err != nil || node.Props["team"] != "support" || node.Props["status"] != "fixed" {
		t.Fatal(node, err)
	}
	ws.Gateway = heldOutGateway{regression: true}
	replay, err = ws.ReviewedReplay(ref, live, nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := replay(r, nil)
	if err != nil || learning.ShouldActivate(report) {
		t.Fatal(report, err)
	}
	stopped := errors.New("budget exhausted")
	replay, err = ws.ReviewedReplay(ref, live, func() error { return stopped })
	if err != nil {
		t.Fatal(err)
	}
	if report, err = replay(r, nil); report != nil || !errors.Is(err, stopped) {
		t.Fatal(report, err)
	}
	unchanged, err := ws.Get(run.ID)
	if err != nil || !reflect.DeepEqual(run, unchanged) {
		t.Fatal("held-out run changed", err)
	}
	live.AddNode(graph.Node{ID: "held-a"})
	if _, err = ws.ReviewedReplay(ref, live, nil); err == nil {
		t.Fatal("overlapping records accepted")
	}
}
