package agents

import (
	"github.com/idristarwala/agent-factory/internal/graph"
	"strings"
	"testing"
)

type panelVote Decision

func (v panelVote) Evaluate(*graph.Mutation, []Rule) (Decision, error) { return Decision(v), nil }

func TestQuorumChecksServedModelIdentity(t *testing.T) {
	for _, tc := range []struct {
		second   string
		approved bool
	}{{"second", true}, {"FIRST", false}, {"", false}} {
		judge := NewQuorumJudge([]Judge{panelVote{Approved: true, Model: "first"}, panelVote{Approved: true, Model: tc.second}}, 2)
		decision, err := judge.Evaluate(&graph.Mutation{}, nil)
		if err != nil || decision.Approved != tc.approved || !strings.Contains(decision.Feedback, "first") {
			t.Fatal(tc, decision, err)
		}
	}
}
