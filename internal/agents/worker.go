package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/llm"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// LLMWorker is a real worker that calls the model gateway (cheap tier only).
type LLMWorker struct {
	Definition AgentVersion
	ID         string
	Role       string
	Gateway    models.Gateway
	Graph      graph.Graph
	Rules      []Rule
	Refresh    RuleRefresher
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
	if uow.InputSubgraph == "" {
		return nil, fmt.Errorf("input record ID is required")
	}
	node, err := w.Graph.GetNode(uow.InputSubgraph)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(node)
	if err != nil {
		return nil, err
	}
	context = string(raw)
	var ruleText strings.Builder
	for _, r := range w.Rules {
		ruleText.WriteString(fmt.Sprintf("- %s\n", renderRule(r)))
	}
	if ruleText.Len() == 0 {
		ruleText.WriteString("(no active rules)\n")
	}

	var familyNote string
	if field := w.Definition.Config["group_field"]; field != "" {
		fam, err := GatherFamily(w.Graph, uow.InputSubgraph, field)
		if err != nil {
			return nil, err
		}
		if fam != nil {
			familyNote = fam.PromptFragment()
		}
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
%s
%s
Propose improvements as a JSON object of attribute key-value pairs.
Respect the rules. Use the gathered evidence. Do not invent facts.
Respond ONLY with valid JSON of the allowed fields.`, uow.Description, ruleText.String(), context, familyNote, evidenceNote)

	// Schema fields come from the unit of work when provided
	allowed, types, err := outputFields(uow, node.Props)
	if err != nil {
		return nil, err
	}
	schema := BuildResponseSchema(allowed, uow.RequiredFields)
	properties := schema["properties"].(map[string]any)
	for k, typ := range types {
		properties[k] = map[string]any{"type": typ}
	}
	if instructions := w.Definition.Config["instructions"]; instructions != "" {
		prompt += "\nWorker instructions: " + instructions
	}

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		resp, err := w.Gateway.Generate(models.Request{
			CallerRole: "worker",
			Tier:       models.TierCheap,
			System:     "You produce structured attribute updates that obey the provided rules and evidence. Output JSON only.",
			User:       prompt, Images: uow.Images,
			Temperature: 0.2,
			MaxTokens:   DefaultWorkerConfig().MaxTokens,
			Schema:      schema,
		})
		if err != nil {
			// Configured protocol clients already apply their transport retry policy.
			var failure *llm.Error
			if errors.As(err, &failure) && failure.Type != "BAD_RESPONSE" {
				return nil, err
			}
			lastErr = err
			time.Sleep(time.Second)
			continue
		}

		updates, err := ParseAndValidate(resp.Content, allowed)
		if err == nil {
			err = validateFields(updates, types, uow.RequiredFields)
		}
		if err != nil {
			LogValidationFailure(resp.Content, err)
			lastErr = fmt.Errorf("worker validation failed: %w; content length=%d", err, len(resp.Content))
			continue
		}

		if !uow.PreserveStatus {
			updates["status"] = "fixed"
		}

		return &graph.Mutation{
			Goal: uow.Description, UnitOfWorkID: uow.ID, Images: uow.Images,
			AgentName: w.Definition.Name, AgentRevision: w.Definition.Revision, RuleRevisions: ruleRevisions(w.Rules),
			Updates: []graph.Node{{
				ID:    uow.InputSubgraph,
				Type:  node.Type,
				Props: updates,
			}},
			Evidence: []string{"source: llm-worker-with-rules", "model: " + resp.Model, "attempt: " + fmt.Sprint(attempt+1)},
		}, nil
	}
	return nil, lastErr
}
