package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/idristarwala/agent-factory/internal/grouping"
	"github.com/idristarwala/agent-factory/internal/schema"
)

type fixture struct {
	Entities []schema.Entity `json:"entities"`
	Labels   []struct {
		ID    string  `json:"id"`
		Group *string `json:"group"`
	} `json:"labels"`
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
	if err := benchmark(data, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func benchmark(data []byte, out io.Writer) error {
	var f fixture
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	if len(f.Entities) == 0 {
		return fmt.Errorf("fixture needs entities")
	}
	ids := make(map[string]bool, len(f.Entities))
	for _, e := range f.Entities {
		if strings.TrimSpace(e.ID) == "" || ids[e.ID] {
			return fmt.Errorf("missing or duplicate entity ID %q", e.ID)
		}
		ids[e.ID] = true
	}

	// Only the evaluator receives these rows. Both detectors receive the source entities.
	var truth []schema.Entity
	if f.Labels != nil {
		for _, label := range f.Labels {
			if !ids[label.ID] || label.Group == nil {
				return fmt.Errorf("unknown, repeated or incomplete label for %q", label.ID)
			}
			delete(ids, label.ID)
			truth = append(truth, schema.Entity{ID: label.ID, Attributes: map[string]any{"group": *label.Group}})
		}
		if len(ids) > 0 {
			return fmt.Errorf("missing independent labels for %d entities", len(ids))
		}
	}
	det := &grouping.SimpleDetector{}
	results := []struct {
		name      string
		proposals []grouping.Proposal
	}{
		{"Existing groups", nil},
		{"SimpleDetector", det.Detect(f.Entities, "group")},
		{"BlockThenDecide (nil-client fallback)", grouping.BlockThenDecide(context.Background(), nil, f.Entities, "group")},
	}
	if f.Labels == nil {
		fmt.Fprintln(out, "No independent labels supplied. Showing proposal counts only.")
	}
	for _, result := range results {
		if f.Labels == nil {
			fmt.Fprintf(out, "%s: proposals=%d\n", result.name, len(result.proposals))
			continue
		}
		m := grouping.ScorePairMetrics(truth, "group", applyProposals(f.Entities, result.proposals))
		fmt.Fprintf(out, "%s: precision=%.2f recall=%.2f f1=%.2f (tp=%d fp=%d fn=%d)\n", result.name, m.Precision, m.Recall, m.F1, m.TruePos, m.FalsePos, m.FalseNeg)
	}
	return nil
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
