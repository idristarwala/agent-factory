package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/idristarwala/agent-factory/internal/grouping"
	"github.com/idristarwala/agent-factory/internal/schema"
)

type fixture struct {
	Entities []schema.Entity `json:"entities"`
}

func main() {
	path := "examples/labelled_grouping.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read fixture: %v\n", err)
		os.Exit(1)
	}
	var f fixture
	if err := json.Unmarshal(data, &f); err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}

	// SimpleDetector path (deterministic)
	det := &grouping.SimpleDetector{}
	props := det.Detect(f.Entities, "group")
	predicted := applyProposals(f.Entities, props)
	m := grouping.ScorePairMetrics(f.Entities, "group", predicted)
	fmt.Printf("SimpleDetector: precision=%.2f recall=%.2f f1=%.2f (tp=%d fp=%d fn=%d)\n",
		m.Precision, m.Recall, m.F1, m.TruePos, m.FalsePos, m.FalseNeg)

	// BlockThenDecide with nil client falls back to SimpleDetector
	ctx := context.Background()
	props2 := grouping.BlockThenDecide(ctx, nil, f.Entities, "group")
	predicted2 := applyProposals(f.Entities, props2)
	m2 := grouping.ScorePairMetrics(f.Entities, "group", predicted2)
	fmt.Printf("BlockThenDecide (nil client): precision=%.2f recall=%.2f f1=%.2f\n",
		m2.Precision, m2.Recall, m2.F1)

	fmt.Println("Demo complete. Wire a real llm.Client for membership decisions.")
}

func applyProposals(entities []schema.Entity, props []grouping.Proposal) map[string]string {
	out := make(map[string]string)
	for _, e := range entities {
		g, _ := e.Attributes["group"].(string)
		out[e.ID] = g
	}
	for _, p := range props {
		out[p.EntityID] = p.ToGroup
	}
	return out
}
