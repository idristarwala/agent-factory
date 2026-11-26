package agents

import (
	"encoding/json"
	"fmt"
	"strings"

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
}

func NewLLMWorker(id, role string, gw models.Gateway, g graph.Graph) *LLMWorker {
	return &LLMWorker{ID: id, Role: role, Gateway: gw, Graph: g}
}

func (w *LLMWorker) Claim(uow *unitofwork.UnitOfWork) error {
	// Claiming is handled by the store; this is a no-op marker.
	return nil
}

func (w *LLMWorker) Execute(uow *unitofwork.UnitOfWork) (*graph.Mutation, error) {
	// Read relevant node if present
	var context string
	if uow.InputSubgraph != "" {
		n, err := w.Graph.GetNode(uow.InputSubgraph)
		if err == nil {
			b, _ := json.Marshal(n)
			context = string(b)
		}
	}

	prompt := fmt.Sprintf(`You are a catalog enrichment worker.
Goal: %s
Current node (JSON): %s

Propose a single improved title and a short description.
Respond ONLY with valid JSON of this form:
{"title":"...","description":"..."}`, uow.Description, context)

	resp, err := w.Gateway.Generate(models.Request{
		Tier:        models.TierCheap,
		System:      "You produce structured catalog improvements. Output JSON only.",
		User:        prompt,
		Temperature: 0.2,
		MaxTokens:   300,
	})
	if err != nil {
		return nil, err
	}

	// Extract JSON from response (handle markdown fences)
	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var out struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return nil, fmt.Errorf("worker JSON parse failed: %w\nraw: %s", err, resp.Content)
	}

	nodeID := uow.InputSubgraph
	if nodeID == "" {
		nodeID = graph.NodeID("product-unknown")
	}

	return &graph.Mutation{
		UnitOfWorkID: uow.ID,
		Updates: []graph.Node{{
			ID:   nodeID,
			Type: "product",
			Props: map[string]any{
				"title":       out.Title,
				"description": out.Description,
			},
		}},
		Evidence: []string{"source: llm-worker", "model: " + resp.Model},
	}, nil
}
