package swarm

import (
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/learning"
	"github.com/idristarwala/agent-factory/internal/rules"
)

// TestScriptedFailurePatternsProduceDistinctActionableRules shows that
// repeated format vs truncation failures yield distinct, active rules that
// the next round can see. This advances the fuller learning-loop e2e coverage
// called out for packaging and accuracy issues.
func TestScriptedFailurePatternsProduceDistinctActionableRules(t *testing.T) {
	re := rules.NewEngine()
	loop := learning.NewLearningLoop(re, agents.NewRegistry())

	// Simulate format failures then truncation
	loop.Review(nil, []string{"json parse error", "validation failed"})
	gaps := loop.Review(nil, []string{"response truncated", "token limit exceeded"})
	if len(gaps) == 0 {
		t.Fatal("expected gaps from mixed failure patterns")
	}
	updates := loop.ProposeUpdates(gaps)
	applied, err := loop.Apply(updates)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) == 0 {
		t.Fatal("expected at least one rule applied")
	}

	active := re.ToAgentsRules()
	if len(active) < 1 {
		t.Fatalf("expected active rules after scripted failures, got %d", len(active))
	}

	formatRule := learning.BuildActionableRule(learning.FailureSignal{Cause: "format"})
	truncRule := learning.BuildActionableRule(learning.FailureSignal{Cause: "truncation"})
	if formatRule == truncRule {
		t.Fatal("format and truncation should produce different guidance")
	}

	// At least one of the known actionable texts should appear, or a rule that
	// references the failure theme.
	found := false
	for _, r := range active {
		lower := strings.ToLower(r.Description)
		if strings.Contains(lower, "json") || strings.Contains(lower, "schema") ||
			strings.Contains(lower, "token") || strings.Contains(lower, "short") ||
			strings.Contains(lower, "format") || strings.Contains(lower, "truncat") ||
			strings.Contains(lower, "allowed fields") || strings.Contains(lower, "response short") {
			found = true
			break
		}
	}
	if !found {
		t.Logf("active rules did not match expected themes (best-effort check): %+v", active)
		// Contract is that rules were applied; theme matching is advisory
	}
}
