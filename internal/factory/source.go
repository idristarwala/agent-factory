package factory

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/idristarwala/agent-factory/internal/mapping"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/schema"
)

type SourceRef struct {
	Mapping  string `json:"mapping"`
	Revision int    `json:"revision"`
}

type SourceInput struct {
	SourceRef
	AllowPartial bool            `json:"allow_partial,omitempty"`
	Payload      json.RawMessage `json:"payload"`
}

type ImportProblem struct {
	Index  int    `json:"index"`
	Reason string `json:"reason"`
}

func (w *Workspace) mappedRecords(input SourceInput) ([]schema.Entity, []ImportProblem, error) {
	if input.Mapping == "" || input.Revision < 1 {
		return nil, nil, fmt.Errorf("supply a mapping id and explicit positive revision")
	}
	docs, err := revision.Documents(w.Store, "source-mapping/"+input.Mapping)
	if err != nil {
		return nil, nil, err
	}
	if input.Revision > len(docs) {
		return nil, nil, fmt.Errorf("source mapping revision not found")
	}
	var m mapping.ApprovedMapping
	if err := json.Unmarshal(docs[input.Revision-1].Data, &m); err != nil {
		return nil, nil, err
	}
	if len(input.Payload) > 1<<20 || !json.Valid(input.Payload) {
		return nil, nil, fmt.Errorf("supply a JSON payload up to 1 MiB")
	}
	var payload any
	dec := json.NewDecoder(bytes.NewReader(input.Payload))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return nil, nil, err
	}
	rows, err := m.ImportRows(payload)
	if err != nil {
		return nil, nil, err
	}
	var records []schema.Entity
	var problems []ImportProblem
	for _, row := range rows {
		if row.Problem != "" {
			if !input.AllowPartial {
				return nil, nil, fmt.Errorf("source row %d: %s", row.Index, row.Problem)
			}
			problems = append(problems, ImportProblem{Index: row.Index, Reason: row.Problem})
			continue
		}
		records = append(records, *row.Record)
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("no importable records: source row %d: %s", problems[0].Index, problems[0].Reason)
	}
	return records, problems, nil
}

// Preserve the legacy numeric representation for existing runs, and the source
// JSON number precision for records imported with a pinned mapping.
func decodeRun(data []byte, r *Run) error {
	var header struct{ Source *SourceRef }
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if header.Source != nil {
		dec.UseNumber()
	}
	return dec.Decode(r)
}
