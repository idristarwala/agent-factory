package mapping

import (
	"encoding/json"
	"testing"
)

func TestRelationshipTargetsAndDrift(t *testing.T) {
	m := SourceMapping{RecordType: "meter", Identity: []string{"/id"}, Fields: map[string]string{"id": "/id"}, Relationships: []RelationshipPath{{Name: "linked", Path: "/targets"}}}
	sample := []any{map[string]any{"id": "one", "targets": []any{json.Number("9007199254740993"), "two", "two"}}}
	rows, err := m.Preview(sample)
	if err != nil || rows[0].Problem != "" {
		t.Fatal(rows, err)
	}
	links := rows[0].Record.Relations
	if len(links) != 2 || links[0].Target != "9007199254740993" || links[1].Type != "linked" {
		t.Fatal(links)
	}
	observed, err := m.Observe(sample)
	if err != nil {
		t.Fatal(err)
	}
	approved := ApprovedMapping{SourceMapping: m, Observed: observed}
	rows, err = approved.ImportRows([]any{map[string]any{"id": "one", "targets": "changed"}, map[string]any{"id": "two", "targets": []any{false}}})
	if err != nil || rows[0].Problem == "" || rows[1].Problem == "" {
		t.Fatal(rows, err)
	}
	if _, err := m.Observe(map[string]any{"id": "one"}); err == nil {
		t.Fatal("approved unobserved relationship path")
	}
	rows, err = m.Preview(map[string]any{"id": "one"})
	if err != nil || rows[0].Record == nil || len(rows[0].Record.Relations) != 0 {
		t.Fatal(rows, err)
	}
}

func TestMappedImagePath(t *testing.T) {
	m := SourceMapping{ImagePath: "/media/urls", RecordType: "inspection", Identity: []string{"/id"}, Fields: map[string]string{"id": "/id"}}
	row := map[string]any{"id": "x", "media": map[string]any{"urls": []any{"https://example.org/a.png", "https://example.org/b.jpg"}}}
	rows, err := m.Preview(row)
	if err != nil || rows[0].Problem != "" || len(rows[0].Record.ImageURLs) != 2 {
		t.Fatal(rows, err)
	}
	observed, err := m.Observe(row)
	if err != nil {
		t.Fatal(err)
	}
	approved := ApprovedMapping{SourceMapping: m, Observed: observed}
	for _, v := range []any{"https://example.org/changed-shape", []any{true}, []any{"file:///etc/passwd"}} {
		row["media"] = map[string]any{"urls": v}
		rows, err := approved.ImportRows(row)
		if err != nil || rows[0].Problem == "" {
			t.Fatal(rows, err)
		}
	}
}
