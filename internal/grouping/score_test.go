package grouping

import (
	"testing"

	"github.com/idristarwala/agent-factory/internal/schema"
)

func TestScorePairMetrics(t *testing.T) {
	entities := []schema.Entity{
		{ID: "a", Attributes: map[string]any{"group": "g1"}},
		{ID: "b", Attributes: map[string]any{"group": "g1"}},
		{ID: "c", Attributes: map[string]any{"group": "g2"}},
		{ID: "d", Attributes: map[string]any{"group": "g2"}},
	}
	// Perfect prediction
	pred := map[string]string{"a": "g1", "b": "g1", "c": "g2", "d": "g2"}
	m := ScorePairMetrics(entities, "group", pred)
	if m.Precision != 1 || m.Recall != 1 || m.F1 != 1 {
		t.Errorf("perfect: got %+v", m)
	}
	if m.TruePos != 2 {
		t.Errorf("expected 2 true pairs, got %d", m.TruePos)
	}

	// One false positive pair (put c with g1)
	pred2 := map[string]string{"a": "g1", "b": "g1", "c": "g1", "d": "g2"}
	m2 := ScorePairMetrics(entities, "group", pred2)
	if m2.Precision >= 1 || m2.Recall >= 1 {
		t.Errorf("imperfect should be <1: %+v", m2)
	}
	if m2.FalsePos == 0 || m2.FalseNeg == 0 {
		t.Errorf("expected errors: %+v", m2)
	}
}
