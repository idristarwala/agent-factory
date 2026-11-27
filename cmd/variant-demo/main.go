package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/configuredgateway"
	"github.com/idristarwala/agent-factory/internal/llm"
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
	live := flag.Bool("live", false, "score configured model membership decisions")
	flag.Parse()
	if flag.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "usage: variant-demo [-live] [fixture.json]")
		os.Exit(2)
	}
	if flag.NArg() == 1 {
		path = flag.Arg(0)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read fixture: %v\n", err)
		os.Exit(1)
	}
	var client llm.Client
	if *live {
		gw, err := configuredgateway.NewFromEnv()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var ok bool
		client, ok = gw.(llm.Client)
		if !ok {
			fmt.Fprintln(os.Stderr, "gateway does not support membership chat")
			os.Exit(1)
		}
	}
	if err := benchmark(data, os.Stdout, client); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func benchmark(data []byte, out io.Writer, client llm.Client) error {
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
	method := "BlockThenDecide (nil-client fallback)"
	var decisions []grouping.Proposal
	if client != nil {
		method = "BlockThenDecide (configured model)"
		var err error
		decisions, err = grouping.DecideMembership(context.Background(), client, f.Entities, "group")
		if err != nil {
			return err
		}
	} else {
		decisions = grouping.BlockThenDecide(context.Background(), nil, f.Entities, "group")
	}
	results := []struct {
		name      string
		proposals []grouping.Proposal
	}{
		{"Existing groups", nil}, {"SimpleDetector", det.Detect(f.Entities, "group")}, {method, decisions},
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
