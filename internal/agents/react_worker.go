package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/tools"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// ReactWorker runs a bounded observe-think-act loop before proposing a mutation.
type ReactWorker struct {
	ID       string
	Role     string
	Gateway  models.Gateway
	Graph    graph.Graph
	Tools    *tools.Registry
	MaxSteps int
}

func NewReactWorker(id, role string, gw models.Gateway, g graph.Graph, tr *tools.Registry) *ReactWorker {
	return &ReactWorker{
		ID: id, Role: role, Gateway: gw, Graph: g, Tools: tr, MaxSteps: 5,
	}
}

func (w *ReactWorker) Claim(uow *unitofwork.UnitOfWork) error { return nil }

func (w *ReactWorker) Execute(uow *unitofwork.UnitOfWork) (*graph.Mutation, error) {
	var contextParts []string
	if uow.InputSubgraph != "" {
		if n, err := w.Graph.GetNode(uow.InputSubgraph); err == nil {
			b, _ := json.Marshal(n)
			contextParts = append(contextParts, "Current node: "+string(b))
		}
	}

	toolSpecs := ""
	if w.Tools != nil {
		for _, spec := range w.Tools.Specs() {
			toolSpecs += fmt.Sprintf("- %s: %s\n", spec.Name, spec.Description)
		}
	}

	var evidence []string
	scratch := strings.Join(contextParts, "\n")

	for step := 0; step < w.MaxSteps; step++ {
		prompt := fmt.Sprintf(`You are a catalog worker in a ReAct loop.
Goal: %s
Context so far:
%s

Available tools:
%s

Reply with ONE of:
1. TOOL: <tool_name> {"arg":"value"}
2. FINAL: {"title":"...","description":"..."}

Only use FINAL when you have enough information.`, uow.Description, scratch, toolSpecs)

		resp, err := w.Gateway.Generate(models.Request{
			Tier:        models.TierCheap,
			System:      "You are a precise agent. Output only TOOL: or FINAL: lines.",
			User:        prompt,
			Temperature: 0.1,
			MaxTokens:   400,
		})
		if err != nil {
			return nil, err
		}

		line := strings.TrimSpace(resp.Content)
		if strings.HasPrefix(line, "FINAL:") {
			jsonPart := strings.TrimSpace(strings.TrimPrefix(line, "FINAL:"))
			var out struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			}
			if err := json.Unmarshal([]byte(jsonPart), &out); err != nil {
				return nil, fmt.Errorf("final JSON: %w", err)
			}
			nodeID := uow.InputSubgraph
			if nodeID == "" {
				nodeID = "product-unknown"
			}
			return &graph.Mutation{
				UnitOfWorkID: uow.ID,
				Updates: []graph.Node{{
					ID:   graph.NodeID(nodeID),
					Type: "product",
					Props: map[string]any{
						"title":       out.Title,
						"description": out.Description,
					},
				}},
				Evidence: evidence,
			}, nil
		}

		if strings.HasPrefix(line, "TOOL:") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "TOOL:"))
			parts := strings.SplitN(rest, " ", 2)
			if len(parts) < 2 {
				scratch += "\n[error] bad TOOL line"
				continue
			}
			toolName := parts[0]
			var args map[string]any
			if err := json.Unmarshal([]byte(parts[1]), &args); err != nil {
				scratch += "\n[error] bad tool args"
				continue
			}
			t, ok := w.Tools.Get(toolName)
			if !ok {
				scratch += "\n[error] unknown tool " + toolName
				continue
			}
			result, err := t.Call(context.Background(), args)
			if err != nil {
				scratch += "\n[tool error] " + err.Error()
				continue
			}
			scratch += fmt.Sprintf("\n[tool %s] %s", toolName, result)
			evidence = append(evidence, "tool:"+toolName)
			continue
		}

		// Unclear; treat as final attempt next step
		scratch += "\n[model] " + line
	}

	return nil, fmt.Errorf("react loop exhausted without FINAL")
}
