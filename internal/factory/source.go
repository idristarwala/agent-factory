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
	Receipt string `json:"sample_receipt,omitempty"`
	SourceRef
	AllowPartial bool            `json:"allow_partial,omitempty"`
	Payload      json.RawMessage `json:"payload"`
}

type ImportProblem struct {
	Index  int    `json:"index"`
	Reason string `json:"reason"`
}

type SourceLineage struct {
	Receipt string `json:"sample_receipt,omitempty"`
	SourceRef
	mapping.RecordLocation
	Feed   string         `json:"feed,omitempty"`
	Import *ImportPageRef `json:"import,omitempty"`
}

type mappedBatch struct {
	editableFields *[]string
	records        []schema.Entity
	problems       []ImportProblem
	lineage        map[string]SourceLineage
}

func (w *Workspace) approvedMapping(input SourceRef) (mapping.ApprovedMapping, error) {
	if input.Mapping == "" || input.Revision < 1 {
		return mapping.ApprovedMapping{}, fmt.Errorf("supply a mapping id and explicit positive revision")
	}
	docs, err := revision.Documents(w.Store, "source-mapping/"+input.Mapping)
	if err != nil {
		return mapping.ApprovedMapping{}, err
	}
	if input.Revision > len(docs) {
		return mapping.ApprovedMapping{}, fmt.Errorf("source mapping revision not found")
	}
	var m mapping.ApprovedMapping
	if err := json.Unmarshal(docs[input.Revision-1].Data, &m); err != nil {
		return mapping.ApprovedMapping{}, err
	}
	return m, nil
}

func (w *Workspace) mappedRecords(factory string, input SourceInput) (mappedBatch, error) {
	m, err := w.approvedMapping(input.SourceRef)
	if err != nil {
		return mappedBatch{}, err
	}
	if len(input.Payload) > 1<<20 || !json.Valid(input.Payload) {
		return mappedBatch{}, fmt.Errorf("supply a JSON payload up to 1 MiB")
	}
	var payload any
	dec := json.NewDecoder(bytes.NewReader(input.Payload))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return mappedBatch{}, err
	}
	feed := ""
	if input.Receipt != "" {
		feed, err = w.verifiedFeed(factory, input.Receipt, payload)
		if err != nil {
			return mappedBatch{}, err
		}
	}
	rows, err := m.ImportRows(payload)
	if err != nil {
		return mappedBatch{}, err
	}
	batch := mappedBatch{editableFields: m.EditableFields, lineage: map[string]SourceLineage{}}
	for _, row := range rows {
		if row.Problem != "" {
			if !input.AllowPartial {
				return mappedBatch{}, fmt.Errorf("source row %d: %s", row.Index, row.Problem)
			}
			batch.problems = append(batch.problems, ImportProblem{Index: row.Index, Reason: row.Problem})
			continue
		}
		batch.records = append(batch.records, *row.Record)
		batch.lineage[row.Record.ID] = SourceLineage{Feed: feed, Receipt: input.Receipt, SourceRef: input.SourceRef, RecordLocation: *row.Location}
	}
	if len(batch.records) == 0 {
		return mappedBatch{}, fmt.Errorf("no importable records: source row %d: %s", batch.problems[0].Index, batch.problems[0].Reason)
	}
	return batch, nil
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
