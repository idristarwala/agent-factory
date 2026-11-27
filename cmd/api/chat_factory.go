package main

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
)

func createChatFactory(ws *factory.Workspace, goal string) (map[string]string, error) {
	if strings.TrimSpace(goal) == "" {
		return map[string]string{"reply": "What should the new factory accomplish?"}, nil
	}
	types := []string{"string", "integer", "number", "boolean", "array", "object"}
	field := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"name", "type"}, "properties": map[string]any{"name": map[string]string{"type": "string"}, "type": map[string]any{"type": "string", "enum": types}}}
	response, err := ws.Gateway.Generate(models.Request{
		Tier: models.TierCheap, User: goal, MaxTokens: 1500,
		System: "Design a minimal factory for this goal. Return a record type and editable output fields. Preserve field names and record type if the user specifies them.",
		Schema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"record_type", "fields"}, "properties": map[string]any{"record_type": map[string]string{"type": "string"}, "fields": map[string]any{"type": "array", "items": field}}},
	})
	if err != nil {
		return nil, err
	}
	usageMu.Lock()
	usageTotals.TokensUsed += response.Usage["total_tokens"]
	usageMu.Unlock()
	var draft struct {
		RecordType string `json:"record_type"`
		Fields     []struct{ Name, Type string }
	}
	if err := json.Unmarshal([]byte(response.Content), &draft); err != nil {
		return nil, err
	}
	spec := factory.Spec{RecordType: strings.TrimSpace(draft.RecordType), Fields: map[string]string{}}
	if spec.RecordType == "" || len(draft.Fields) == 0 {
		return nil, errors.New("factory design needs a record type and editable fields")
	}
	for _, f := range draft.Fields {
		if strings.TrimSpace(f.Name) == "" || spec.Fields[f.Name] != "" || !slices.Contains(types, f.Type) {
			return nil, errors.New("factory design returned an invalid field")
		}
		spec.Fields[f.Name] = f.Type
	}
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	if _, err := revision.SaveDocument(ws.Store, "factory/"+id, 0, "steward", data); err != nil {
		return nil, err
	}
	return map[string]string{"factory": id, "definition": string(data), "reply": "Created factory " + id + " at revision 1. Add source records of type " + spec.RecordType + " in Goal builder, then send Start followed by your goal."}, nil
}
