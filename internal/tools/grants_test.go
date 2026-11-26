package tools

import (
	"testing"
)

func TestCheckGrant(t *testing.T) {
	grants := []Grant{{
		Pack:        "catalog",
		Tools:       []string{"search", "propose"},
		SideEffects: []string{"read-only", "reversible"},
	}}

	if !CheckGrant(grants, "catalog", "search", "read-only") {
		t.Error("expected grant to allow search")
	}
	if CheckGrant(grants, "catalog", "delete", "irreversible") {
		t.Error("expected grant to deny delete")
	}
}
