package grouping

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/idristarwala/agent-factory/internal/llm"
	"github.com/idristarwala/agent-factory/internal/schema"
)

// MembershipDecision is the LLM's answer for one candidate neighbourhood.
type MembershipDecision struct {
	EntityID string `json:"entity_id"`
	GroupID  string `json:"group_id"`
	Same     bool   `json:"same_product"`
	Reason   string `json:"reason"`
}

// BlockThenDecide builds candidate neighbourhoods with a cheap signature,
// then asks the LLM to decide membership within each neighbourhood.
// Falls back to SimpleDetector if the client is nil or the call fails.
func BlockThenDecide(ctx context.Context, client llm.Client, entities []schema.Entity, groupField string) []Proposal {
	if client == nil {
		return (&SimpleDetector{}).Detect(entities, groupField)
	}

	// Build neighbourhoods by signature (same as SimpleDetector)
	buckets := map[string][]schema.Entity{}
	for _, e := range entities {
		sig := signature(e)
		buckets[sig] = append(buckets[sig], e)
	}

	var all []Proposal
	for _, bucket := range buckets {
		if len(bucket) < 2 {
			continue
		}
		// Ask the model which items belong together
		decisions, err := askMembership(ctx, client, bucket)
		if err != nil || len(decisions) == 0 {
			// Fallback to signature majority
			all = append(all, (&SimpleDetector{}).Detect(bucket, groupField)...)
			continue
		}
		// Convert decisions into proposals (items that the model says should move)
		for _, d := range decisions {
			if !d.Same {
				continue
			}
			for _, e := range bucket {
				if e.ID == d.EntityID {
					cur, _ := e.Attributes[groupField].(string)
					if cur == "" {
						cur = "__ungrouped__"
					}
					if cur != d.GroupID && d.GroupID != "" {
						all = append(all, Proposal{
							EntityID:   e.ID,
							FromGroup:  cur,
							ToGroup:    d.GroupID,
							Reason:     d.Reason,
							Confidence: 0.8,
						})
					}
					break
				}
			}
		}
	}
	return all
}

func askMembership(ctx context.Context, client llm.Client, bucket []schema.Entity) ([]MembershipDecision, error) {
	var lines []string
	for _, e := range bucket {
		brand, _ := e.Attributes["brand"].(string)
		title, _ := e.Attributes["title"].(string)
		lines = append(lines, fmt.Sprintf("- id=%s brand=%q title=%q", e.ID, brand, title))
	}
	prompt := fmt.Sprintf(`Given these candidate items that share a coarse signature, decide which belong to the same product variant group.
Reply with a JSON array of objects: [{"entity_id":"...","group_id":"shared-id-or-new","same_product":true,"reason":"..."}]
Items:
%s`, strings.Join(lines, "\n"))

	resp, err := client.Chat(ctx, llm.ChatRequest{
		Model:    "cheap",
		Messages: []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens: 800,
	})
	if err != nil {
		return nil, err
	}
	var decisions []MembershipDecision
	if err := json.Unmarshal([]byte(resp.Content), &decisions); err != nil {
		// Try to extract a JSON array from the reply
		start := strings.Index(resp.Content, "[")
		end := strings.LastIndex(resp.Content, "]")
		if start >= 0 && end > start {
			_ = json.Unmarshal([]byte(resp.Content[start:end+1]), &decisions)
		}
	}
	return decisions, nil
}
