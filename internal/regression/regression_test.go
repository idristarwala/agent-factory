package regression

import (
	"testing"

	"github.com/idristarwala/agent-factory/internal/graph"
)

// TestApplyDoesNotLoseAttributes is a regression for the data-loss bug
// reported in issue #10.
func TestApplyDoesNotLoseAttributes(t *testing.T) {
	g := graph.NewMemoryGraph()
	g.AddNode(graph.Node{
		ID:   "item-1",
		Type: "product",
		Props: map[string]any{
			"brand":       "Adidas",
			"product_type": "Shoes",
			"color":       "Black",
			"title":       "Old Title",
		},
	})

	mutation := &graph.Mutation{
		Updates: []graph.Node{{
			ID: "item-1",
			Props: map[string]any{
				"title": "Better Title",
			},
		}},
	}
	if err := g.Apply(mutation); err != nil {
		t.Fatal(err)
	}

	n, _ := g.GetNode("item-1")
	if n.Props["brand"] != "Adidas" {
		t.Error("brand was lost")
	}
	if n.Props["product_type"] != "Shoes" {
		t.Error("product_type was lost")
	}
	if n.Props["color"] != "Black" {
		t.Error("color was lost")
	}
	if n.Props["title"] != "Better Title" {
		t.Error("title was not updated")
	}
}
