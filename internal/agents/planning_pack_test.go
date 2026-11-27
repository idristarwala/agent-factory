package agents

import (
	"errors"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/pack"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func TestExplicitPlanningPack(t *testing.T) {
	steward := NewSmartSteward(unitofwork.NewMemoryStore())
	units, err := steward.InterpretGoal("review catalog", nil)
	if err != nil || len(units) != 1 || !strings.Contains(units[0].Description, "[general]") {
		t.Fatal(units, err)
	}
	steward.PlanningPack, err = pack.Parse("pack.yaml", []byte("name: support\nworker_roles: [triage, review]\n"))
	if err != nil {
		t.Fatal(err)
	}
	units, err = steward.InterpretGoal("review catalog", nil)
	if err != nil || len(units) != 2 {
		t.Fatal(units, err)
	}
	for i, role := range []string{"triage", "review"} {
		if !strings.Contains(units[i].Description, "[support]") || !strings.HasSuffix(units[i].Description, role) {
			t.Fatal(units[i].Description)
		}
	}
}

type scopeGateway struct {
	properties map[string]any
	reply      string
}

func (g *scopeGateway) Generate(req models.Request) (*models.Response, error) {
	g.properties = req.Schema["properties"].(map[string]any)
	return &models.Response{Content: g.reply}, nil
}

func TestGraphlessPlanExecutesWithConfiguredScope(t *testing.T) {
	s := NewSmartSteward(unitofwork.NewMemoryStore())
	s.PlanningPack = &pack.Pack{Name: "review", WorkerRoles: []string{"editor", "reviewer"}}
	s.AllowedFields = []string{"title"}
	units, err := s.InterpretGoal("review the notice", nil)
	if err != nil || len(units) != 2 {
		t.Fatal(units, err)
	}
	if len(units[1].AllowedFields) != 1 {
		t.Fatalf("missing role scope: %v", units[1].AllowedFields)
	}
	s.AllowedFields[0] = "other"
	units[1].AllowedFields[0] = "source"
	u := units[0]
	u.InputSubgraph = "notice"
	g := graph.NewMemoryGraph()
	g.AddNode(graph.Node{ID: "notice", Props: map[string]any{"title": "draft", "source": "opening hours"}})
	gw := &scopeGateway{reply: `{"title":"Opening hours"}`}
	w := NewLLMWorker("editor", "worker", gw, g, nil)
	m, err := w.Execute(u)
	if err != nil {
		t.Fatal(err)
	}
	if len(gw.properties) != 1 || gw.properties["title"] == nil || m.Updates[0].Props["title"] != "Opening hours" {
		t.Fatal(gw.properties, m)
	}
	gw.reply = `{"source":"different hours"}`
	if _, err := w.Execute(u); !errors.Is(err, ErrInvalidPatch) {
		t.Fatalf("unexpected result for out-of-scope edit: %v", err)
	}
	s.AllowedFields = nil
	units, err = s.InterpretGoal("review the notice", nil)
	if err != nil {
		t.Fatal(err)
	}
	units[0].InputSubgraph = "notice"
	gw.properties = nil
	if _, err := w.Execute(units[0]); err == nil || gw.properties != nil {
		t.Fatal("ambiguous unit reached gateway", err)
	}
}
