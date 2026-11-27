package grouping

import (
	"sort"
	"strings"

	"github.com/idristarwala/agent-factory/internal/schema"
)

// GroupKey is how entities should be grouped.
// Catalog: variant group. Pricing: price family. Inventory: location cluster.
type GroupKey struct {
	Name   string
	Fields []string
	Value  map[string]any
}

// Proposal is a suggested regrouping.
type Proposal struct {
	EntityID   string
	FromGroup  string
	ToGroup    string
	Reason     string
	Confidence float64
}

// Detector finds entities that appear to be in the wrong group.
type Detector interface {
	Detect(entities []schema.Entity, groupField string) []Proposal
}

// SimpleDetector uses attribute similarity.
// Deterministic: ties broken by sorted group name, ungrouped items are first-class.
type SimpleDetector struct{}

func (d *SimpleDetector) Detect(entities []schema.Entity, groupField string) []Proposal {
	buckets := map[string][]schema.Entity{}
	for _, e := range entities {
		sig := signature(e)
		buckets[sig] = append(buckets[sig], e)
	}

	var proposals []Proposal
	for _, bucket := range buckets {
		if len(bucket) < 2 {
			continue
		}
		counts := map[string]int{}
		for _, e := range bucket {
			g, _ := e.Attributes[groupField].(string)
			if g == "" {
				g = "__ungrouped__"
			}
			counts[g]++
		}

		// Deterministic majority: highest count, then lexicographic name
		var best string
		var bestN int
		keys := make([]string, 0, len(counts))
		for k := range counts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, g := range keys {
			n := counts[g]
			if n > bestN || (n == bestN && (best == "" || g < best)) {
				best, bestN = g, n
			}
		}

		for _, e := range bucket {
			g, _ := e.Attributes[groupField].(string)
			if g == "" {
				g = "__ungrouped__"
			}
			if g != best && best != "__ungrouped__" {
				proposals = append(proposals, Proposal{
					EntityID:   e.ID,
					FromGroup:  g,
					ToGroup:    best,
					Reason:     "shares signature with majority group",
					Confidence: float64(bestN) / float64(len(bucket)),
				})
			}
		}
	}
	// Stable order for determinism across runs
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].EntityID < proposals[j].EntityID
	})
	return proposals
}

var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true,
	"of": true, "for": true, "with": true, "to": true, "in": true,
}

func signature(e schema.Entity) string {
	brand, _ := e.Attributes["brand"].(string)
	title, _ := e.Attributes["title"].(string)
	ptype, _ := e.Attributes["product_type"].(string)
	if brand == "" && title == "" && ptype == "" {
		return "id:" + e.ID
	}
	stem := nameStem(strings.ToLower(title))
	return strings.ToLower(brand) + "|" + strings.ToLower(ptype) + "|" + stem
}

func nameStem(title string) string {
	words := strings.Fields(title)
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?\"'")
		if w != "" && !stopWords[w] && len(w) > 2 {
			return w
		}
	}
	if len(words) > 0 {
		return words[0]
	}
	return ""
}
