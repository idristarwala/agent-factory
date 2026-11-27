package agents

import (
	"fmt"
	"strings"

	"github.com/idristarwala/agent-factory/internal/graph"
)

// QuorumJudge requires a minimum number of independent judges to approve
// a mutation before it is accepted. Disagreement is recorded so an Arbiter
// or human can escalate.
//
// This is the practical quality lever from the consensus research:
// a single judge cannot land a bad change.
type QuorumJudge struct {
	Judges     []Judge
	MinApprove int // quorum size (default 2)
}

func NewQuorumJudge(judges []Judge, minApprove int) *QuorumJudge {
	if minApprove < 1 {
		minApprove = 1
	}
	if minApprove > len(judges) {
		minApprove = len(judges)
	}
	return &QuorumJudge{Judges: judges, MinApprove: minApprove}
}

func (q *QuorumJudge) Evaluate(proposal *graph.Mutation, rules []Rule) (Decision, error) {
	if len(q.Judges) == 0 {
		return Decision{Approved: false, Reason: "no judges configured"}, nil
	}

	var approvals int
	seen := map[string]bool{}
	distinct := true
	var reasons []string

	for i, j := range q.Judges {
		dec, err := j.Evaluate(proposal, rules)
		if err != nil {
			reasons = append(reasons, fmt.Sprintf("judge-%d error: %v", i, err))
			continue
		}
		model := strings.ToLower(strings.TrimSpace(dec.Model))
		if len(q.Judges) > 1 && (model == "" || seen[model]) {
			distinct = false
		}
		seen[model] = true
		if dec.Approved {
			approvals++
		}
		reasons = append(reasons, fmt.Sprintf("judge-%d %s: %s", i, dec.Model, dec.Reason))
	}

	approved := distinct && approvals >= q.MinApprove
	reason := fmt.Sprintf("quorum %d/%d (min %d)", approvals, len(q.Judges), q.MinApprove)
	if !distinct {
		reason += "; missing or duplicate judge model identity"
	}
	if !approved {
		reason = "escalation needed: " + reason
	}

	return Decision{
		Approved: approved,
		Reason:   reason,
		Feedback: fmt.Sprintf("%v", reasons),
	}, nil
}
