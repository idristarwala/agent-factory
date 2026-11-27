package revision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// Document is a snapshot of a declarative definition. Data contains configuration,
// including references to external secrets when a definition requires credentials.
type Document struct {
	ID        string          `json:"id"`
	Version   int             `json:"version"`
	CreatedBy string          `json:"created_by"`
	ChangedAt time.Time       `json:"changed_at"`
	Data      json.RawMessage `json:"data"`
}

func Documents(s Store, id string) ([]Document, error) {
	rows, err := s.Read(Config, id)
	if err != nil {
		return nil, err
	}
	out := make([]Document, 0, len(rows))
	for _, row := range rows {
		var d Document
		if err := json.Unmarshal(row.Data, &d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}
func SaveDocument(s Store, id string, expected int, actor string, data json.RawMessage) (Document, error) {
	if !json.Valid(data) {
		return Document{}, fmt.Errorf("invalid JSON definition")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var fields map[string]any
	if err := decoder.Decode(&fields); err != nil || fields == nil {
		return Document{}, fmt.Errorf("definition must be an object")
	}
	canonical, err := json.Marshal(fields)
	if err != nil {
		return Document{}, err
	}
	previous, err := Documents(s, id)
	if err != nil {
		return Document{}, err
	}
	if len(previous) != expected {
		return Document{}, ErrConflict
	}
	if expected > 0 && bytes.Equal(previous[expected-1].Data, canonical) {
		return previous[expected-1], nil
	}
	d := Document{ID: id, Version: expected + 1, CreatedBy: actor, ChangedAt: time.Now().UTC(), Data: canonical}
	payload, err := json.Marshal(d)
	if err != nil {
		return Document{}, err
	}
	if err := s.Insert(Config, Snapshot{ID: id, Version: d.Version, Data: payload}); err != nil {
		return Document{}, err
	}
	return d, nil
}
