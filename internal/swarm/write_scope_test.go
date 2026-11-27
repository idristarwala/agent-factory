package swarm

import (
	"encoding/json"
	"fmt"
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
	reject          bool
	t               *testing.T
}

func (g *scopeGateway) Generate(req models.Request) (*models.Response, error) {
	if req.CallerRole == "judge" {
		g.judges++
		if g.reject {
			return &models.Response{Content: "REJECT: unsupported"}, nil
		}
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
		{"qualified_other_domain", "Fix ticket assignments", "assignment", `{"assignment":"IT"}`, nil, 1, 1},
		{"qualified_outside_grant", "Fix ticket assignments", "", "", []string{"assigned_team"}, 0, 0},
		{"explicit_scope", "Assign the support team from the routing guide", "assigned_team", `{"assigned_team":"IT"}`, []string{"assigned_team"}, 1, 1},
		{"ambiguous", "Review this record", "", "", nil, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := graph.NewMemoryGraph()
			g.AddNode(graph.Node{ID: "record", Type: "ticket", Props: map[string]any{"title": "old", "clothing_size": "M", "actual_color": "blue", "assigned_team": "unknown", "assignment": "unknown", "status": "open"}})
			gw := &scopeGateway{field: tc.field, reply: tc.reply, t: t}
			steward := agents.NewSmartSteward(unitofwork.NewMemoryStore())
			steward.SetRuntime(gw, g, nil)
			steward.AllowedFields = tc.configured
			r := NewContinuousRunner(steward, agents.NewLLMJudge(gw, g), queue.NewMemoryQueue(), g, rules.NewEngine(), nil)
			r.Workers = 1
			r.Budget.MaxRounds = 1
			r.Budget.MaxFailures = 1
			err := r.Run(t.Context(), tc.goal)
			if (err == nil) != (tc.judges == 1) || gw.workers != tc.workers || gw.judges != tc.judges {
				t.Fatal(err, gw)
			}
			for _, entry := range r.Audit.All() {
				if entry.Action == "proposed" {
					var m graph.Mutation
					if err := json.Unmarshal(entry.Proposal, &m); err != nil {
						t.Fatal(err)
					}
					if m.CompleteOnApply != "record" || len(m.Updates) != 1 || len(m.Updates[0].Props) != 1 || m.Updates[0].Props[tc.field] == nil {
						t.Fatal("audit contains a different patch", m)
					}
				}
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
			if tc.judges == 1 && (n.Props[tc.field] != "IT" || n.Props["status"] != "fixed") {
				t.Fatal(n)
			}
		})
	}
}

func TestCompletionRequiresAppliedApproval(t *testing.T) {
	for _, tc := range []struct {
		name, field     string
		reject, observe bool
	}{
		{"approved_asset", "condition", false, false},
		{"observed_asset", "condition", false, true},
		{"rejected_ticket", "priority", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := graph.NewMemoryGraph()
			g.AddNode(graph.Node{ID: "case", Props: map[string]any{tc.field: "old", "source_note": "unchanged", "status": "open"}})
			gw := &scopeGateway{field: tc.field, reply: fmt.Sprintf(`{"%s":"new"}`, tc.field), reject: tc.reject, t: t}
			steward := agents.NewSmartSteward(unitofwork.NewMemoryStore())
			steward.AllowedFields = []string{tc.field}
			steward.SetRuntime(gw, g, nil)
			r := NewContinuousRunner(steward, agents.NewLLMJudge(gw, g), queue.NewMemoryQueue(), g, rules.NewEngine(), nil)
			r.Workers = 1
			r.Budget.MaxRounds = 1
			r.SetObserve(tc.observe)
			err := r.Run(t.Context(), "Correct the field using the source")
			applied := !tc.reject && !tc.observe
			if (err == nil) != applied || gw.judges != 1 {
				t.Fatal(err, gw.judges)
			}
			node, err := g.GetNode("case")
			if err != nil {
				t.Fatal(err)
			}
			value, status := "old", "open"
			if applied {
				value, status = "new", "fixed"
			}
			if node.Props[tc.field] != value || node.Props["status"] != status || node.Props["source_note"] != "unchanged" {
				t.Fatal(node)
			}
		})
	}
}
