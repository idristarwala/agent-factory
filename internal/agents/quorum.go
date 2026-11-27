package agents

import (
	"fmt"

	"github.com/idristarwala/agent-factory/internal/graph"
)

// QuorumJudge requires a minimum number of independent judges to approve
// a mutation before it is accepted. Disagreement is recorded so an Arbiter
// or human can escalate.
//
// This is the practical quality lever from the consensus research:
// a single judge cannot land a bad change.
type QuorumJudge struct {
	Judges []Judge
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

	var approvals, rejections int
	var reasons []string

	for i, j := range q.Judges {
		dec, err := j.Evaluate(proposal, rules)
		if err != nil {
			reasons = append(reasons, fmt.Sprintf("judge-%d error: %v", i, err))
			continue
		}
		if dec.Approved {
			approvals++
		} else {
			rejections++
		}
		reasons = append(reasons, fmt.Sprintf("judge-%d: %s", i, dec.Reason))
	}

	approved := approvals >= q.MinApprove
	reason := fmt.Sprintf("quorum %d/%d (min %d)", approvals, len(q.Judges), q.MinApprove)
	if !approved {
		reason = "escalation needed: " + reason
	}

	return Decision{
		Approved: approved,
		Reason:   reason,
		Feedback: fmt.Sprintf("%v", reasons),
	}, nil
}
