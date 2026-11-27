package main

import (
	"encoding/json"
	"errors"
	"github.com/idristarwala/agent-factory/internal/artifact"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
)

func createChatFactory(ws *factory.Workspace, goal string, refs []artifact.Ref, evidence artifact.Evidence, proposalID string, expected int) (map[string]any, error) {
	if strings.TrimSpace(goal) == "" {
		return map[string]any{"reply": "What should the new factory accomplish?"}, nil
	}
	id := proposalID
	if id == "" {
		if expected != 0 {
			return nil, errors.New("invalid draft revision")
		}
		id = uuid.NewString()
	} else {
		if uuid.Validate(id) != nil || expected < 1 || len(refs) == 0 {
			return nil, errors.New("select the draft and its attachments to revise it")
		}
		approved, err := revision.Documents(ws.Store, "factory/"+id)
		if err != nil {
			return nil, err
		}
		if len(approved) > 0 {
			return nil, errors.New("this proposal is already approved")
		}
		history, err := revision.Documents(ws.Store, "goal-proposal/"+id)
		if err != nil {
			return nil, err
		}
		if len(history) != expected || history[expected-1].CreatedBy != "steward" {
			return nil, revision.ErrConflict
		}
		goal += "\nRevise this earlier draft using the current sources. Preserve the record type and exact list of field names and types unless the reviewer requests a schema change:\n" + string(history[expected-1].Data)
	}
	types := []string{"string", "integer", "number", "boolean", "array", "object"}
	field := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"name", "type"}, "properties": map[string]any{"name": map[string]string{"type": "string"}, "type": map[string]any{"type": "string", "enum": types}}}
	props := map[string]any{"record_type": map[string]string{"type": "string"}, "fields": map[string]any{"type": "array", "items": field}}
	required := []string{"record_type", "fields"}
	instructions := "Design a minimal factory for this goal. Treat attachments as source material for the requested design. Return a record type and editable output fields. Preserve field names and record type if the user specifies them."
	if len(refs) > 0 {
		props["goal"] = map[string]string{"type": "string"}
		props["criteria"] = map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"text": map[string]string{"type": "string"}, "sources": map[string]any{"type": "array", "items": map[string]string{"type": "integer"}}}, "required": []string{"text", "sources"}, "additionalProperties": false}}
		required = append(required, "goal", "criteria")
		instructions += " Propose a concise goal and 1 to 8 verifiable acceptance criteria grounded in the supplied sources. For each criterion include the numeric Source labels that support it."
	}
	response, err := ws.Gateway.Generate(models.Request{
		CallerRole: "steward", Tier: models.TierCheap, User: goal + evidence.Text, Images: evidence.Images, MaxTokens: 1500,
		System: instructions,
		Schema: map[string]any{"type": "object", "additionalProperties": false, "required": required, "properties": props},
	})
	if err != nil {
		return nil, err
	}
	usageMu.Lock()
	usageTotals.TokensUsed += response.Usage["total_tokens"]
	usageMu.Unlock()
	var draft struct {
		Goal     string
		Criteria []struct {
			Text    string
			Sources []int
		}
		RecordType string `json:"record_type"`
		Fields     []struct{ Name, Type string }
	}
	if err := json.Unmarshal([]byte(response.Content), &draft); err != nil {
		return nil, err
	}
	spec := factory.Spec{Artifacts: refs, RecordType: strings.TrimSpace(draft.RecordType), Fields: map[string]string{}}
	if spec.RecordType == "" || len(draft.Fields) == 0 {
		return nil, errors.New("factory design needs a record type and editable fields")
	}
	for _, f := range draft.Fields {
		if strings.TrimSpace(f.Name) == "" || spec.Fields[f.Name] != "" || !slices.Contains(types, f.Type) {
			return nil, errors.New("factory design returned an invalid field")
		}
		spec.Fields[f.Name] = f.Type
	}
	if len(refs) > 0 {
		spec.Goal = strings.TrimSpace(draft.Goal)
		if spec.Goal == "" || len(draft.Criteria) == 0 || len(draft.Criteria) > 8 {
			return nil, errors.New("steward returned an incomplete goal proposal")
		}
		for _, item := range draft.Criteria {
			c := factory.Criterion{Text: strings.TrimSpace(item.Text)}
			if c.Text == "" || len(item.Sources) == 0 {
				return nil, errors.New("criterion needs text and at least one citation")
			}
			for _, n := range item.Sources {
				if n < 1 || n > len(evidence.Citations) {
					return nil, errors.New("steward cited an unknown source location")
				}
				citation := evidence.Citations[n-1]
				if !slices.Contains(c.Citations, citation) {
					c.Citations = append(c.Citations, citation)
				}
			}
			spec.Criteria = append(spec.Criteria, c)
		}
	}
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}

	if len(refs) > 0 {
		proposal, err := revision.SaveDocument(ws.Store, "goal-proposal/"+id, expected, "steward", data)
		if err != nil {
			return nil, err
		}
		return map[string]any{"proposal": id, "version": proposal.Version, "design": spec, "reply": "Review the proposed goal and its cited acceptance criteria."}, nil
	}
	if _, err := revision.SaveDocument(ws.Store, "factory/"+id, 0, "steward", data); err != nil {
		return nil, err
	}
	return map[string]any{"factory": id, "definition": string(data), "reply": "Created factory " + id + " at revision 1. Add source records of type " + spec.RecordType + " in Goal builder, then send Start followed by your goal."}, nil
}
