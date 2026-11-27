package grouping

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
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
	proposals, _ := decideBuckets(ctx, client, entities, groupField, true)
	return proposals
}

// DecideMembership evaluates model decisions without substituting offline predictions on errors.
func DecideMembership(ctx context.Context, client llm.Client, entities []schema.Entity, groupField string) ([]Proposal, error) {
	if client == nil {
		return nil, fmt.Errorf("live membership requires a gateway")
	}
	return decideBuckets(ctx, client, entities, groupField, false)
}

func decideBuckets(ctx context.Context, client llm.Client, entities []schema.Entity, groupField string, fallback bool) ([]Proposal, error) {
	if client == nil {
		return (&SimpleDetector{}).Detect(entities, groupField), nil
	}
	buckets := map[string][]schema.Entity{}
	for _, e := range entities {
		key := signature(e)
		buckets[key] = append(buckets[key], e)
	}
	var all []Proposal
	for _, key := range slices.Sorted(maps.Keys(buckets)) {
		bucket := buckets[key]
		if len(bucket) < 2 {
			continue
		}
		slices.SortFunc(bucket, func(a, b schema.Entity) int { return cmp.Compare(a.ID, b.ID) })
		proposals, err := askMembership(ctx, client, bucket, groupField)
		if err != nil {
			if !fallback {
				return nil, fmt.Errorf("membership for bucket starting at %q: %w", bucket[0].ID, err)
			}
			proposals = (&SimpleDetector{}).Detect(bucket, groupField)
		}
		all = append(all, proposals...)
	}
	slices.SortFunc(all, func(a, b Proposal) int { return cmp.Compare(a.EntityID, b.EntityID) })
	return all, nil
}

func askMembership(ctx context.Context, client llm.Client, bucket []schema.Entity, groupField string) ([]Proposal, error) {
	data, err := json.Marshal(bucket)
	if err != nil {
		return nil, err
	}
	prompt := fmt.Sprintf("Decide membership within these candidate records using their source attributes. Current groups are in %q. Return one decision per ID. Reuse an existing group ID when it fits; use a distinct ID for a new group. For a record with no suitable assignment set same_product to false. JSON records:\n%s", groupField, data)
	fields := map[string]any{"entity_id": map[string]any{"type": "string"}, "group_id": map[string]any{"type": "string"}, "same_product": map[string]any{"type": "boolean"}, "reason": map[string]any{"type": "string"}}
	item := map[string]any{"type": "object", "properties": fields, "required": []string{"entity_id", "group_id", "same_product", "reason"}, "additionalProperties": false}
	contract := map[string]any{"type": "object", "properties": map[string]any{"decisions": map[string]any{"type": "array", "items": item}}, "required": []string{"decisions"}, "additionalProperties": false}
	zero := 0.0
	resp, err := client.Chat(ctx, llm.ChatRequest{Model: "cheap", CallerRole: llm.RoleWorker, Messages: []llm.Message{{Role: "user", Content: prompt}}, Temperature: &zero, MaxTokens: 800, ResponseSchema: contract})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("gateway returned no response")
	}
	var result struct {
		Decisions []struct {
			EntityID string  `json:"entity_id"`
			GroupID  *string `json:"group_id"`
			Same     *bool   `json:"same_product"`
			Reason   *string `json:"reason"`
		} `json:"decisions"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return nil, fmt.Errorf("invalid membership JSON: %w", err)
	}
	current := make(map[string]string, len(bucket))
	for _, e := range bucket {
		current[e.ID], _ = e.Attributes[groupField].(string)
	}
	var proposals []Proposal
	for _, d := range result.Decisions {
		old, ok := current[d.EntityID]
		if !ok || d.Same == nil || d.GroupID == nil || d.Reason == nil {
			return nil, fmt.Errorf("unknown, duplicate or incomplete decision for %q", d.EntityID)
		}
		delete(current, d.EntityID)
		if !*d.Same {
			continue
		}
		if strings.TrimSpace(*d.GroupID) == "" || *d.GroupID == "__ungrouped__" {
			return nil, fmt.Errorf("missing assigned group for %q", d.EntityID)
		}
		if old == "" {
			old = "__ungrouped__"
		}
		if old != *d.GroupID {
			proposals = append(proposals, Proposal{EntityID: d.EntityID, FromGroup: old, ToGroup: *d.GroupID, Reason: *d.Reason, Confidence: 0.8})
		}
	}
	if len(current) > 0 {
		return nil, fmt.Errorf("missing decisions for %d records", len(current))
	}
	return proposals, nil
}
