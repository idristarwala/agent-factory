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
type LLMWorker struct {
	ID      string
	Role    string
	Gateway models.Gateway
	Graph   graph.Graph
	Rules   []Rule
	Refresh RuleRefresher
}

type RuleRefresher func() []Rule

func NewLLMWorker(id, role string, gw models.Gateway, g graph.Graph, rules []Rule) *LLMWorker {
	return &LLMWorker{ID: id, Role: role, Gateway: gw, Graph: g, Rules: rules}
}

func (w *LLMWorker) SetRefresher(fn RuleRefresher) { w.Refresh = fn }

func (w *LLMWorker) Claim(uow *unitofwork.UnitOfWork) error { return nil }

func (w *LLMWorker) Execute(uow *unitofwork.UnitOfWork) (*graph.Mutation, error) {
	if w.Refresh != nil {
		w.Rules = w.Refresh()
	}

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

	var familyNote string
	if uow.InputSubgraph != "" {
		fam, err := GatherFamily(w.Graph, uow.InputSubgraph, "group")
		if err == nil && fam != nil {
			familyNote = fam.PromptFragment()
		}
	}
	if familyNote == "" {
		familyNote = "If this item has siblings, apply consistent wording across the family."
	}

	var evidenceNote string
	if len(uow.Evidence) > 0 {
		evidenceNote = "Gathered evidence:\n" + strings.Join(uow.Evidence, "\n") + "\n"
	} else {
		evidenceNote = "(no additional evidence gathered)\n"
	}

	prompt := fmt.Sprintf(`You are a specialized worker fixing entities.
Goal / unit of work: %s
Active rules you MUST follow:
%s
Current entity (JSON): %s
Original title: %s
Original description: %s
%s
%s
Propose improvements as a JSON object of attribute key-value pairs.
Respect the rules. Use the gathered evidence. Do not invent facts.
Respond ONLY with valid JSON of the allowed fields.`, uow.Description, ruleText.String(), context, originalTitle, originalDesc, familyNote, evidenceNote)

	// Schema fields come from the unit of work when provided
	allowed := uow.AllowedFields
	required := uow.RequiredFields
	if len(allowed) == 0 {
		allowed = []string{"title", "description"}
	}
	if len(required) == 0 {
		required = []string{"title"}
	}
	schema := BuildResponseSchema(allowed, required)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		resp, err := w.Gateway.Generate(models.Request{
			Tier:        models.TierCheap,
			System:      "You produce structured attribute updates that obey the provided rules and evidence. Output JSON only.",
			User:        prompt,
			Temperature: 0.2,
			MaxTokens:   8000,
			Schema:      schema,
		})
		if err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}

		updates, err := ParseAndValidate(resp.Content, allowed)
		if err != nil {
			lastErr = fmt.Errorf("worker validation failed: %w; content length=%d", err, len(resp.Content))
			continue
		}

		updates["status"] = "fixed"

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
