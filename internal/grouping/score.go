package grouping

import (
	"github.com/idristarwala/agent-factory/internal/schema"
)

// PairMetrics is precision/recall/F1 over entity pairs that share a group.
// Used to score a detector or applied proposals against a labelled set.
type PairMetrics struct {
	Precision float64
	Recall    float64
	F1        float64
	TruePos   int
	FalsePos  int
	FalseNeg  int
}

// ScorePairMetrics compares predicted groups to labelled groups.
// Groups are the value of groupField on each entity.
// Ungrouped / empty groups are ignored for pair counting.
func ScorePairMetrics(entities []schema.Entity, groupField string, predicted map[string]string) PairMetrics {
	// predicted[entityID] = groupID (after applying proposals or detector output)
	// Build true pairs and pred pairs.
	truePairs := make(map[[2]string]struct{})
	predPairs := make(map[[2]string]struct{})

	byTrue := map[string][]string{}
	byPred := map[string][]string{}
	for _, e := range entities {
		g, _ := e.Attributes[groupField].(string)
		if g != "" && g != "__ungrouped__" {
			byTrue[g] = append(byTrue[g], e.ID)
		}
		if p, ok := predicted[e.ID]; ok && p != "" && p != "__ungrouped__" {
			byPred[p] = append(byPred[p], e.ID)
		}
	}

	addPairs := func(groups map[string][]string, dest map[[2]string]struct{}) {
		for _, ids := range groups {
			for i := 0; i < len(ids); i++ {
				for j := i + 1; j < len(ids); j++ {
					a, b := ids[i], ids[j]
					if a > b {
						a, b = b, a
					}
					dest[[2]string{a, b}] = struct{}{}
				}
			}
		}
	}
	addPairs(byTrue, truePairs)
	addPairs(byPred, predPairs)

	tp, fp, fn := 0, 0, 0
	for p := range predPairs {
		if _, ok := truePairs[p]; ok {
			tp++
		} else {
			fp++
		}
	}
	for p := range truePairs {
		if _, ok := predPairs[p]; !ok {
			fn++
		}
	}

	var prec, rec, f1 float64
	if tp+fp > 0 {
		prec = float64(tp) / float64(tp+fp)
	}
	if tp+fn > 0 {
		rec = float64(tp) / float64(tp+fn)
	}
	if prec+rec > 0 {
		f1 = 2 * prec * rec / (prec + rec)
	}
	return PairMetrics{Precision: prec, Recall: rec, F1: f1, TruePos: tp, FalsePos: fp, FalseNeg: fn}
}
