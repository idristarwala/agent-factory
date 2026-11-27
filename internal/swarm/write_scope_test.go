package swarm

import (
	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/queue"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
	"testing"
)

type scopeGateway struct {
	field, reply    string
	workers, judges int
	t               *testing.T
}

func (g *scopeGateway) Generate(req models.Request) (*models.Response, error) {
	if req.CallerRole == "judge" {
		g.judges++
		return &models.Response{Content: "APPROVE: fixture"}, nil
	}
	g.workers++
	props := req.Schema["properties"].(map[string]any)
	if len(props) != 1 || props[g.field] == nil {
		g.t.Errorf("unexpected write scope %v", props)
	}
	return &models.Response{Content: g.reply}, nil
}

func TestContinuousWriteScope(t *testing.T) {
	for _, tc := range []struct {
		name, goal, field, reply string
		configured               []string
		workers, judges          int
	}{
		{"title_only", "Edit only title", "title", `{"title":"fixed","clothing_size":"XL","actual_color":"purple"}`, nil, 2, 0},
		{"explicit_scope", "Assign the support team from the routing guide", "assigned_team", `{"assigned_team":"IT"}`, []string{"assigned_team"}, 1, 1},
		{"ambiguous", "Review this record", "", "", nil, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := graph.NewMemoryGraph()
			g.AddNode(graph.Node{ID: "record", Props: map[string]any{"title": "old", "clothing_size": "M", "actual_color": "blue", "assigned_team": "unknown", "status": "open"}})
			gw := &scopeGateway{field: tc.field, reply: tc.reply, t: t}
			steward := agents.NewSmartSteward(unitofwork.NewMemoryStore())
			steward.SetRuntime(gw, g, nil)
			steward.AllowedFields = tc.configured
			r := NewContinuousRunner(steward, agents.NewLLMJudge(gw, g), queue.NewMemoryQueue(), g, rules.NewEngine(), nil)
			r.Workers = 1
			r.Budget.MaxRounds = 2
			r.Budget.MaxFailures = 1
			err := r.Run(t.Context(), tc.goal)
			if (err == nil) != (tc.judges == 1) || gw.workers != tc.workers || gw.judges != tc.judges {
				t.Fatal(err, gw)
			}
			n, err := g.GetNode("record")
			if err != nil {
				t.Fatal(err)
			}
			if n.Props["title"] != "old" || n.Props["clothing_size"] != "M" || n.Props["actual_color"] != "blue" {
				t.Fatal(n)
			}
			if tc.judges == 0 && n.Props["status"] != "open" {
				t.Fatal(n)
			}
			if tc.judges == 1 && (n.Props["assigned_team"] != "IT" || n.Props["status"] != "fixed") {
				t.Fatal(n)
			}
		})
	}
}
