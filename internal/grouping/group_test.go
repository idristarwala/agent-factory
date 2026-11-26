package grouping

import (
	"testing"

	"github.com/idristarwala/agent-factory/internal/schema"
)

func TestSimpleDetectorDeterministic(t *testing.T) {
	entities := []schema.Entity{
		{ID: "1", Attributes: map[string]any{"brand": "Acme", "title": "Widget Red", "product_type": "tool", "group": "g1"}},
		{ID: "2", Attributes: map[string]any{"brand": "Acme", "title": "Widget Blue", "product_type": "tool", "group": "g2"}},
		{ID: "3", Attributes: map[string]any{"brand": "Acme", "title": "Widget Green", "product_type": "tool", "group": "g1"}},
		{ID: "4", Attributes: map[string]any{"brand": "Other", "title": "Gadget", "product_type": "tool", "group": ""}},
	}

	d := &SimpleDetector{}
	p1 := d.Detect(entities, "group")
	p2 := d.Detect(entities, "group")

	if len(p1) != len(p2) {
		t.Fatalf("non-deterministic length: %d vs %d", len(p1), len(p2))
	}
	for i := range p1 {
		if p1[i].EntityID != p2[i].EntityID || p1[i].ToGroup != p2[i].ToGroup {
			t.Errorf("run mismatch at %d", i)
		}
	}

	// Ungrouped item should be considered (signature may not match others)
	foundUngrouped := false
	for _, p := range p1 {
		if p.EntityID == "4" {
			foundUngrouped = true
		}
	}
	// Not required to propose for unmatched signature; just ensure no panic
	t.Logf("ungrouped proposed: %v, total proposals: %d", foundUngrouped, len(p1))
}

func TestSignatureIncludesType(t *testing.T) {
	e := schema.Entity{
		ID: "x",
		Attributes: map[string]any{
			"brand":        "Acme",
			"product_type": "Tool",
			"title":        "Super Widget",
		},
	}
	sig := signature(e)
	if sig != "acme|tool|super" {
		t.Errorf("unexpected signature %q", sig)
	}
}
