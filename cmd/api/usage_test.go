package main

import (
	"github.com/idristarwala/agent-factory/internal/factory"
	"testing"
)

func TestRunUsageAggregateAndFilter(t *testing.T) {
	runs := []factory.Run{
		{ID: "a", Calls: []factory.Call{{Input: 9, Output: 3, Total: 12, Cost: 0.5, Priced: true}, {Failed: true}}, Tools: map[string]int{"read_goal_artifacts": 1}},
		{ID: "b", Calls: []factory.Call{{Input: 3, Output: 2, Total: 5}}, Tools: map[string]int{"collect_mapped_images": 2, "read_goal_artifacts": 1}},
		{ID: "legacy"},
	}
	all := runUsage(runs, "")
	if all.Input != 12 || all.Output != 5 || all.Tokens != 17 || all.EstimatedUSD != 0.5 || all.Unpriced != 1 || all.Failed != 1 || all.ModelCalls != 3 || all.ToolCalls != 4 || all.Tools["read_goal_artifacts"] != 2 {
		t.Fatal(all)
	}
	b := runUsage(runs, "b")
	if b.ToolCalls != 3 || b.Tokens != 5 || b.ModelCalls != 1 || b.Unpriced != 1 || b.EstimatedUSD != 0 {
		t.Fatal(b)
	}
}
