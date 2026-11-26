package agents

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// LLMWorker is a domain-agnostic worker that calls the model gateway (cheap tier).
type LLMWorker struct {
	ID      string
	Role    string
	Gateway models.Gateway
	Graph   graph.Graph
	Rules   *rules.Engine
}

func NewLLMWorker(id, role string, gw models.Gateway, g graph.Graph) *LLMWorker {
	return &LLMWorker{ID: id, Role: role, Gateway: gw, Graph: g}
}

func (w *LLMWorker) SetRules(re *rules.Engine) {
	w.Rules = re
}

func (w *LLMWorker) Claim(uow *unitofwork.UnitOfWork) error {
	return nil
}

func (w *LLMWorker) Execute(uow *unitofwork.UnitOfWork) (*graph.Mutation, error) {
	var context string
	if uow.InputSubgraph != "" {
		n, err := w.Graph.GetNode(uow.InputSubgraph)
		if err == nil {
			b, _ := json.Marshal(n)
			context = string(b)
		}
	}

	var rulesText string
	if w.Rules != nil {
		var descs []string
		for _, r := range w.Rules.ToAgentsRules() {
			descs = append(descs, r.Description)
		}
		rulesText = strings.Join(descs, "\n- ")
	}

	prompt := fmt.Sprintf(`You are a specialized agent in a domain-agnostic factory.
Goal / task: %s
Active rules you must follow:
- %s
Current entity (JSON): %s

Propose updates to any attributes that help achieve the goal while respecting the rules.
Respond ONLY with valid JSON object of key-value pairs to set or update, e.g.:
{"title": "improved title", "description": "...", "pt": "Footwear"}
Do not include keys you are not changing. Output pure JSON only.`, uow.Description, rulesText, context)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		resp, err := w.Gateway.Generate(models.Request{
			Tier:        models.TierCheap,
			System:      "You produce structured improvements. Output JSON object only. No markdown.",
			User:        prompt,
			Temperature: 0.2,
			MaxTokens:   600,
		})
		if err != nil {
			lastErr = err
			continue
		}

		content := strings.TrimSpace(resp.Content)
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)

		var updates map[string]any
		if err := json.Unmarshal([]byte(content), &updates); err != nil {
			lastErr = fmt.Errorf("JSON parse failed: %w\nraw: %s", err, resp.Content)
			continue
		}

		nodeID := uow.InputSubgraph
		if nodeID == "" {
			nodeID = graph.NodeID("entity-unknown")
		}

		return &graph.Mutation{
			UnitOfWorkID: uow.ID,
			Updates: []graph.Node{{
				ID:    nodeID,
				Type:  "entity",
				Props: updates,
			}},
			Evidence: []string{"source: generic-llm-worker", "model: " + resp.Model, "role: " + w.Role},
		}, nil
	}
	return nil, lastErr
}
