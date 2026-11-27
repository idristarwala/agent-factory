package agents

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
)

// LLMJudge evaluates proposals using rules.
// Fails closed: any model error or unclear reply rejects the proposal.
// Now receives the full proposed mutation so it can see facts and original context.
type LLMJudge struct {
	Definition AgentVersion
	Gateway    models.Gateway
	Graph      graph.Graph
}

func NewLLMJudge(gw models.Gateway, g graph.Graph) *LLMJudge {
	return &LLMJudge{Gateway: gw, Graph: g}
}

func (j *LLMJudge) Evaluate(proposal *graph.Mutation, rules []Rule) (Decision, error) {
	var ruleText strings.Builder
	for _, r := range rules {
		ruleText.WriteString(fmt.Sprintf("- %s\n", renderRule(r)))
	}
	if ruleText.Len() == 0 {
		ruleText.WriteString("(no rules)\n")
	}

	var proposalJSON string
	if b, err := json.Marshal(proposal); err == nil {
		proposalJSON = string(b)
	}

	var original string
	if len(proposal.Updates) > 0 && j.Graph != nil {
		if n, err := j.Graph.GetNode(proposal.Updates[0].ID); err == nil {
			if b, err := json.Marshal(n); err == nil {
				original = string(b)
			}
		}
	}

	prompt := fmt.Sprintf(`Evaluate this proposal against the rules and the original entity.
Rules:
%s
Original entity (JSON):
%s
Proposed mutation (JSON):
%s

Check that the proposal does not invent facts or violate rules.
Reply with exactly one line:
APPROVE: <reason>
or
REJECT: <reason>`, ruleText.String(), original, proposalJSON)

	if extra := j.Definition.Config["instructions"]; extra != "" {
		prompt += "\nJudge instructions: " + extra
	}
	resp, err := j.Gateway.Generate(models.Request{
		Tier:        models.TierFrontier,
		System:      "You are a strict judge. Output only APPROVE: or REJECT: followed by a short reason.",
		User:        prompt,
		Temperature: 0.0,
		MaxTokens:   300,
	})
	if err != nil {
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
	return Decision{Approved: false, Reason: "unclear response, fail closed: " + line}, nil
}
