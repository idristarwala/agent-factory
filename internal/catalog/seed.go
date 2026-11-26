package catalog

import "github.com/idristarwala/agent-factory/internal/graph"

// Seed returns a multi-PT, multi-brand catalog with intentional quality issues.
// Includes items that should be variants of each other.

func Seed() []graph.Node {
	return []graph.Node{
		// --- Apparel / Nike ---
		{
			ID:   "nike-shirt-blue-s",
			Type: "product",
			Props: map[string]any{
				"pt":          "Apparel",
				"brand":       "Nike",
				"title":       "Blue Shirt", // bad
				"description": "A shirt",    // bad
				"color":       "blue",
				"size":        "S",
				"material":    "cotton", // right
				"group":       "nike-basic-tee",
			},
		},
		{
			ID:   "nike-shirt-blue-m",
			Type: "product",
			Props: map[string]any{
				"pt":          "Apparel",
				"brand":       "Nike",
				"title":       "Nike Tee M", // mediocre
				"description": "Nike t-shirt medium",
				"color":       "blue",
				"size":        "M",
				"material":    "", // missing
				"group":       "nike-basic-tee",
			},
		},
		{
			ID:   "nike-shirt-red-m",
			Type: "product",
			Props: map[string]any{
				"pt":          "Apparel",
				"brand":       "Nike",
				"title":       "Red Nike T-Shirt Medium",
				"description": "Comfortable red Nike t-shirt in medium. 100% cotton.",
				"color":       "red",
				"size":        "M",
				"material":    "cotton",
				"group":       "nike-basic-tee",
			},
		},

		// --- Footwear / Adidas ---
		{
			ID:   "adidas-shoe-black-10",
			Type: "product",
			Props: map[string]any{
				"pt":          "Footwear",
				"brand":       "Adidas",
				"title":       "Black Shoe", // bad
				"description": "Shoe",       // bad
				"color":       "black",
				"size":        "10",
				"material":    "synthetic",
				"gender":      "mens",
				"group":       "adidas-runner",
			},
		},
		{
			ID:   "adidas-shoe-black-11",
			Type: "product",
			Props: map[string]any{
				"pt":          "Footwear",
				"brand":       "Adidas",
				"title":       "Adidas Running Shoe",
				"description": "Running shoe for men",
				"color":       "black",
				"size":        "11",
				"material":    "", // missing
				"gender":      "mens",
				"group":       "adidas-runner",
			},
		},

		// --- Electronics / Sony ---
		{
			ID:   "sony-headphones-wh1000",
			Type: "product",
			Props: map[string]any{
				"pt":            "Electronics",
				"brand":         "Sony",
				"title":         "Sony WH-1000XM5 Wireless Noise Cancelling Headphones",
				"description":   "Industry-leading noise cancellation with premium sound and long battery life.",
				"color":         "black",
				"connectivity":  "bluetooth",
				"battery_hours": "30",
				"group":         "sony-wh1000xm5",
			},
		},
		{
			ID:   "sony-headphones-wh1000-silver",
			Type: "product",
			Props: map[string]any{
				"pt":            "Electronics",
				"brand":         "Sony",
				"title":         "Headphones", // bad
				"description":   "Nice headphones", // bad
				"color":         "silver",
				"connectivity":  "", // missing
				"battery_hours": "30",
				"group":         "sony-wh1000xm5",
			},
		},

		// --- Wrong attribute example ---
		{
			ID:   "nike-shoe-wrong-attr",
			Type: "product",
			Props: map[string]any{
				"pt":            "Footwear",
				"brand":         "Nike",
				"title":         "Nike Air Max",
				"description":   "Running shoe",
				"color":         "white",
				"size":          "9",
				"material":      "leather",
				"battery_hours": "10", // wrong for footwear
				"group":         "nike-airmax",
			},
		},
	}
}

// Groups that should be treated as variants
func VariantGroups() map[string][]string {
	return map[string][]string{
		"nike-basic-tee": {"nike-shirt-blue-s", "nike-shirt-blue-m", "nike-shirt-red-m"},
		"adidas-runner":  {"adidas-shoe-black-10", "adidas-shoe-black-11"},
		"sony-wh1000xm5": {"sony-headphones-wh1000", "sony-headphones-wh1000-silver"},
	}
}
