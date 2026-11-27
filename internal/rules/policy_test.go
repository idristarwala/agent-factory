package rules

import (
	"fmt"
	"testing"
)

func TestCustomerPolicyImportIsActiveAndVersioned(t *testing.T) {
	engine := NewEngine()
	policies := []Policy{{ID: "desk/owner", Name: "Assignment", Description: "Use the verified routing guide", Source: "customer"}}
	for i := 1; i <= 27; i++ {
		policies = append(policies, ParseSimpleTree(fmt.Sprintf("if category is %d then route to team %d", i, i), "routing guide")...)
	}
	imported := IngestPolicies(engine, policies, "customer")
	active := engine.ToAgentsRules()
	if len(imported) != 28 || len(engine.Active()) != 28 || len(active) != 28 {
		t.Fatalf("imported %d, active %d, agent-visible %d", len(imported), len(engine.Active()), len(active))
	}
	if imported[0].ID != policies[0].ID {
		t.Fatalf("lost stable customer policy ID %s", imported[0].ID)
	}
	byID := map[string]bool{}
	for _, r := range active {
		byID[r.ID] = true
	}
	for i, r := range imported {
		if !byID[r.ID] || r.CreatedBy != "customer" || r.Version != 1 || r.Condition != policies[i].Condition || r.Action != policies[i].Action {
			t.Fatal(r)
		}
	}
	policies[0].Description = "Use the updated routing guide"
	revised := IngestPolicies(engine, policies[:1], "operator")[0]
	history := engine.History("desk/owner")
	if revised.Version != 2 || len(history) != 2 || history[0].Description != "Use the verified routing guide" || history[1].CreatedBy != "operator" || len(engine.ToAgentsRules()) != 28 {
		t.Fatal(revised, history)
	}
	engine.Propose(VersionedRule{Description: "Unreviewed learned rule", CreatedBy: "steward"})
	if len(engine.ToAgentsRules()) != 28 {
		t.Fatal("inactive proposal reached agents")
	}
}
