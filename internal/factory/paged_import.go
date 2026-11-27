package factory

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/idristarwala/agent-factory/internal/mapping"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/schema"
)

type ImportPageRef struct {
	ID   string `json:"id"`
	Page int    `json:"page"`
}

type PageSource interface {
	Sample(context.Context, string, string, map[string]string) (json.RawMessage, error)
}
type ImportSpec struct {
	SourceRef
	Feed         string `json:"feed"`
	CursorParam  string `json:"cursor_param"`
	NextPath     string `json:"next_path"`
	AllowPartial bool   `json:"allow_partial,omitempty"`
}
type ImportPage struct {
	Lineage  map[string]SourceLineage `json:"lineage,omitempty"`
	Number   int                      `json:"page"`
	Spec     ImportSpec               `json:"spec"`
	Next     string                   `json:"next,omitempty"`
	Complete bool                     `json:"complete"`
	Records  []schema.Entity          `json:"records"`
	Problems []ImportProblem          `json:"problems,omitempty"`
}

func importKey(factory, id string) string {
	return "source-import/" + base64.RawURLEncoding.EncodeToString([]byte(factory)) + "/" + base64.RawURLEncoding.EncodeToString([]byte(id))
}
func (w *Workspace) ImportPages(factory, id string) ([]ImportPage, error) {
	docs, err := revision.Documents(w.Store, importKey(factory, id))
	if err != nil {
		return nil, err
	}
	pages := make([]ImportPage, len(docs))
	for i, d := range docs {
		dec := json.NewDecoder(bytes.NewReader(d.Data))
		dec.UseNumber()
		if err := dec.Decode(&pages[i]); err != nil {
			return nil, err
		}
	}
	return pages, nil
}
func (w *Workspace) AdvanceImport(ctx context.Context, source PageSource, factory, id string, spec ImportSpec, expected, limit int) ([]ImportPage, error) {
	if factory == "" || id == "" || spec.Feed == "" || spec.Mapping == "" || spec.Revision < 1 || spec.CursorParam == "" || !strings.HasPrefix(spec.NextPath, "/") || expected < 0 || limit < 1 || limit > 10 {
		return nil, fmt.Errorf("supply a factory, import id, feed, pinned mapping, cursor parameter, next-token pointer and 1–10 pages")
	}
	pages, err := w.ImportPages(factory, id)
	if err != nil {
		return nil, err
	}
	if len(pages) != expected {
		return nil, revision.ErrConflict
	}
	cursor := ""
	cursors := map[string]bool{}
	ids := map[string]bool{}
	for _, p := range pages {
		if p.Spec != spec {
			return nil, fmt.Errorf("import specification differs from its checkpoint")
		}
		if p.Complete {
			return []ImportPage{}, nil
		}
		cursor = p.Next
		cursors[cursor] = true
		for _, r := range p.Records {
			ids[r.ID] = true
		}
	}
	docs, err := revision.Documents(w.Store, "source-mapping/"+spec.Mapping)
	if err != nil {
		return nil, err
	}
	if spec.Revision > len(docs) {
		return nil, fmt.Errorf("unknown mapping revision")
	}
	var approved mapping.ApprovedMapping
	if err = json.Unmarshal(docs[spec.Revision-1].Data, &approved); err != nil {
		return nil, err
	}
	committed := []ImportPage{}
	for i := 0; i < limit; i++ {
		if err := ctx.Err(); err != nil {
			return committed, err
		}
		query := map[string]string{}
		if cursor != "" {
			query[spec.CursorParam] = cursor
		}
		raw, err := source.Sample(ctx, factory, spec.Feed, query)
		if err != nil {
			return committed, err
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var payload any
		if err := dec.Decode(&payload); err != nil {
			return committed, fmt.Errorf("invalid JSON source page")
		}
		token, ok, err := mapping.SourceValue(payload, spec.NextPath)
		if err != nil || !ok {
			return committed, fmt.Errorf("next-token pointer is invalid or missing")
		}
		next := ""
		if token != nil {
			var valid bool
			next, valid = token.(string)
			if !valid {
				return committed, fmt.Errorf("next-token value must be a string or null")
			}
		}
		if next != "" && cursors[next] {
			return committed, fmt.Errorf("source repeated a pagination cursor")
		}
		page := ImportPage{Lineage: map[string]SourceLineage{}, Number: expected + 1, Spec: spec, Next: next, Complete: next == "", Records: []schema.Entity{}}
		data, ok, err := mapping.SourceValue(payload, approved.Collection)
		if err != nil || !ok {
			return committed, fmt.Errorf("source collection is missing")
		}
		array, isArray := data.([]any)
		if !isArray || len(array) > 0 {
			rows, err := approved.ImportRows(payload)
			if err != nil {
				return committed, err
			}
			for _, row := range rows {
				if row.Problem == "" && ids[row.Record.ID] {
					row.Problem = "identity already committed on an earlier page"
				}
				if row.Problem != "" {
					if !spec.AllowPartial {
						return committed, fmt.Errorf("page %d row %d: %s", page.Number, row.Index, row.Problem)
					}
					page.Problems = append(page.Problems, ImportProblem{Index: row.Index, Reason: row.Problem})
					continue
				}
				page.Records = append(page.Records, *row.Record)
				page.Lineage[row.Record.ID] = SourceLineage{SourceRef: spec.SourceRef, RecordLocation: *row.Location, Feed: spec.Feed, Import: &ImportPageRef{ID: id, Page: page.Number}}
				ids[row.Record.ID] = true
			}
		}
		encoded, err := json.Marshal(page)
		if err != nil {
			return committed, err
		}
		if _, err = revision.SaveDocument(w.Store, importKey(factory, id), expected, "importer", encoded); err != nil {
			return committed, err
		}
		committed = append(committed, page)
		expected++
		if page.Complete {
			break
		}
		cursor = next
		cursors[next] = true
	}
	return committed, nil
}
