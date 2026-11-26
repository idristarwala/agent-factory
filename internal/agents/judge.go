package agents

import "github.com/idristarwala/agent-factory/internal/graph"

// StubJudge always approves for early scaffolding.
// Real implementation will evaluate against rules and evidence.

type StubJudge struct{}

func NewStubJudge() *StubJudge {
	return &StubJudge{}
}

func (j *StubJudge) Evaluate(proposal *graph.Mutation, rules []Rule) (Decision, error) {
	return Decision{
		Approved: true,
		Reason:   "stub approval — no rules evaluated yet",
		Feedback: "",
	}, nil
}
