package agents

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// LLMWorker is a real worker that calls the model gateway (cheap tier only).
// Receives active rules and retries structured JSON parsing.
type LLMWorker struct {
	ID      string
	Role    string
	Gateway models.Gateway
	Graph   graph.Graph
	Rules   []Rule
}

func NewLLMWorker(id, role string, gw models.Gateway, g graph.Graph, rules []Rule) *LLMWorker {
	return &LLMWorker{ID: id, Role: role, Gateway: gw, Graph: g, Rules: rules}
}

func (w *LLMWorker) Claim(uow *unitofwork.UnitOfWork) error {
	return nil
}

func (w *LLMWorker) Execute(uow *unitofwork.UnitOfWork) (*graph.Mutation, error) {
	var context string
	var originalTitle, originalDesc string
	if uow.InputSubgraph != "" {
		n, err := w.Graph.GetNode(uow.InputSubgraph)
		if err == nil {
			b, _ := json.Marshal(n)
			context = string(b)
			if t, ok := n.Props["title"].(string); ok {
				originalTitle = t
			}
			if d, ok := n.Props["description"].(string); ok {
				originalDesc = d
			}
		}
	}

	var ruleText strings.Builder
	for _, r := range w.Rules {
		ruleText.WriteString(fmt.Sprintf("- %s\n", r.Description))
	}
	if ruleText.Len() == 0 {
		ruleText.WriteString("(no active rules)\n")
	}

	// Basic family context note (siblings can be added by domain pack later)
	familyNote := "If this item has siblings, apply consistent wording across the family."

	prompt := fmt.Sprintf(`You are a specialized worker fixing entities.
Goal / unit of work: %s
Active rules you MUST follow:
%s
Current entity (JSON): %s
Original title: %s
Original description: %s
%s

Propose improvements as a JSON object of attribute key-value pairs.
Respect the rules (e.g. keep original wording when required).
Respond ONLY with valid JSON of this form:
{"title":"...", "description":"...", "other_attr":"..."}`, uow.Description, ruleText.String(), context, originalTitle, originalDesc, familyNote)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		resp, err := w.Gateway.Generate(models.Request{
			Tier:        models.TierCheap,
			System:      "You produce structured attribute updates that obey the provided rules. Output JSON only.",
			User:        prompt,
			Temperature: 0.2,
			MaxTokens:   800,
		})
		if err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}

		content := strings.TrimSpace(resp.Content)
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)

		var updates map[string]any
		if err := json.Unmarshal([]byte(content), &updates); err != nil {
			lastErr = fmt.Errorf("worker JSON parse failed: %w\nraw: %s", err, resp.Content)
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
			Evidence: []string{"source: llm-worker-with-rules", "model: " + resp.Model, "attempt: " + fmt.Sprint(attempt+1)},
		}, nil
	}
	return nil, lastErr
}
