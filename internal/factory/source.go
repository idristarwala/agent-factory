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
	Payload json.RawMessage `json:"payload"`
}

func (w *Workspace) mappedRecords(input SourceInput) ([]schema.Entity, error) {
	if input.Mapping == "" || input.Revision < 1 {
		return nil, fmt.Errorf("supply a mapping id and explicit positive revision")
	}
	docs, err := revision.Documents(w.Store, "source-mapping/"+input.Mapping)
	if err != nil {
		return nil, err
	}
	if input.Revision > len(docs) {
		return nil, fmt.Errorf("source mapping revision not found")
	}
	var m mapping.SourceMapping
	if err := json.Unmarshal(docs[input.Revision-1].Data, &m); err != nil {
		return nil, err
	}
	if len(input.Payload) > 1<<20 || !json.Valid(input.Payload) {
		return nil, fmt.Errorf("supply a JSON payload up to 1 MiB")
	}
	var payload any
	dec := json.NewDecoder(bytes.NewReader(input.Payload))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	rows, err := m.Preview(payload)
	if err != nil {
		return nil, err
	}
	records := make([]schema.Entity, len(rows))
	for i, row := range rows {
		if row.Problem != "" {
			return nil, fmt.Errorf("source row %d: %s", i, row.Problem)
		}
		records[i] = *row.Record
	}
	return records, nil
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
