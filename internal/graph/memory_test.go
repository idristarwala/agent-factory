package graph

import (
	"testing"
)

func TestApplyMergesProps(t *testing.T) {
	g := NewMemoryGraph()
	g.AddNode(Node{
		ID:   "sku-1",
		Type: "product",
		Props: map[string]any{
			"brand": "Nike",
			"pt":    "Footwear",
			"title": "Blue Shoe",
		},
	})

	err := g.Apply(&Mutation{
		Updates: []Node{{
			ID:   "sku-1",
			Props: map[string]any{
				"title": "Nike Blue Running Shoe",
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	n, err := g.GetNode("sku-1")
	if err != nil {
		t.Fatal(err)
	}
	if n.Props["brand"] != "Nike" {
		t.Errorf("brand lost: got %v", n.Props["brand"])
	}
	if n.Props["pt"] != "Footwear" {
		t.Errorf("pt lost: got %v", n.Props["pt"])
	}
	if n.Props["title"] != "Nike Blue Running Shoe" {
		t.Errorf("title not updated: got %v", n.Props["title"])
	}
}
