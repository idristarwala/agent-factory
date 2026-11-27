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
}

func TestUngroupedItemsPlaced(t *testing.T) {
	entities := []schema.Entity{
		{ID: "1", Attributes: map[string]any{"brand": "Acme", "title": "Super Widget Red", "product_type": "tool", "group": "g1"}},
		{ID: "2", Attributes: map[string]any{"brand": "Acme", "title": "Super Widget Blue", "product_type": "tool", "group": "g1"}},
		{ID: "3", Attributes: map[string]any{"brand": "Acme", "title": "Super Widget Green", "product_type": "tool", "group": ""}}, // ungrouped, same signature
	}

	d := &SimpleDetector{}
	proposals := d.Detect(entities, "group")

	found := false
	for _, p := range proposals {
		if p.EntityID == "3" && p.ToGroup == "g1" && p.FromGroup == "__ungrouped__" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ungrouped item 3 to be proposed into g1, got %+v", proposals)
	}
}

func TestSignatureIncludesTypeAndStem(t *testing.T) {
	e := schema.Entity{
		ID: "x",
		Attributes: map[string]any{
			"brand":        "Acme",
			"product_type": "Tool",
			"title":        "The Super Widget",
		},
	}
	sig := signature(e)
	if sig != "acme|tool|super" {
		t.Errorf("unexpected signature %q", sig)
	}
}
