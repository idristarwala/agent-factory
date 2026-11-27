package agents

import (
	"strings"
	"testing"

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
