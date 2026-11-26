package agents

import (
	"fmt"
	"strings"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
)

// LLMJudge evaluates proposals using rules.
// Fails closed: any model error or unclear reply rejects the proposal.
type LLMJudge struct {
	Gateway models.Gateway
}

func NewLLMJudge(gw models.Gateway) *LLMJudge {
	return &LLMJudge{Gateway: gw}
}

func (j *LLMJudge) Evaluate(proposal *graph.Mutation, rules []Rule) (Decision, error) {
	var ruleText strings.Builder
	for _, r := range rules {
		ruleText.WriteString(fmt.Sprintf("- %s\n", r.Description))
	}
	if ruleText.Len() == 0 {
		ruleText.WriteString("(no rules)\n")
	}

	var title, desc string
	if len(proposal.Updates) > 0 {
		if t, ok := proposal.Updates[0].Props["title"].(string); ok {
			title = t
		}
		if d, ok := proposal.Updates[0].Props["description"].(string); ok {
			desc = d
		}
	}

	prompt := fmt.Sprintf(`Evaluate this proposal against the rules.
Rules:
%s
Proposed title: %s
Proposed description: %s

Reply with exactly one line:
APPROVE: <reason>
or
REJECT: <reason>`, ruleText.String(), title, desc)

	resp, err := j.Gateway.Generate(models.Request{
		Tier:        models.TierFrontier,
		System:      "You are a strict judge. Output only APPROVE: or REJECT: followed by a short reason.",
		User:        prompt,
		Temperature: 0.0,
		MaxTokens:   150,
	})
	if err != nil {
		// Fail closed
		return Decision{Approved: false, Reason: "gateway error, fail closed: " + err.Error()}, nil
	}

	line := strings.TrimSpace(resp.Content)
	upper := strings.ToUpper(line)
	if strings.HasPrefix(upper, "APPROVE") {
		return Decision{Approved: true, Reason: line}, nil
	}
	if strings.HasPrefix(upper, "REJECT") {
		return Decision{Approved: false, Reason: line}, nil
	}
	// Unclear reply also fails closed
	return Decision{Approved: false, Reason: "unclear response, fail closed: " + line}, nil
}
