package factory

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/store"
	"path/filepath"
	"testing"
)

type testFeed func(map[string]string) (json.RawMessage, error)

func (f testFeed) Sample(ctx context.Context, factory, name string, q map[string]string) (json.RawMessage, error) {
	return f(q)
}
func TestResumePagedImport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.sqlite")
	db, err := store.NewSQLStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ws := Workspace{Store: db.Definitions()}
	spec := ImportSpec{SourceRef: SourceRef{Mapping: "records", Revision: 1}, Feed: "feed", CursorParam: "after", NextPath: "/next", AllowPartial: true}
	for key, data := range map[string]string{"factory/desk": `{"record_type":"case","fields":{"team":"string"}}`, "source-mapping/records": `{"collection":"/rows","record_type":"case","identity":["/id"],"fields":{"team":"/team"}}`} {
		if _, err := revision.SaveDocument(ws.Store, key, 0, "human", []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	var calls int
	feed := testFeed(func(q map[string]string) (json.RawMessage, error) {
		calls++
		if calls == 1 && len(q) == 0 {
			return json.RawMessage(`{"rows":[{"id":9007199254740993,"team":"A"}],"next":"page2"}`), nil
		}
		if q["after"] != "page2" {
			t.Fatal("unexpected cursor", q)
		}
		if calls == 2 {
			return nil, errors.New("unavailable")
		}
		return json.RawMessage(`{"rows":[{"id":9007199254740993,"team":"B"},{"id":2,"team":"C"}],"next":null}`), nil
	})
	pages, err := ws.AdvanceImport(t.Context(), feed, "desk", "batch", spec, 0, 2)
	if err == nil || len(pages) != 1 {
		t.Fatal(pages, err)
	}
	db.Close()
	db, err = store.NewSQLStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ws.Store = db.Definitions()
	if _, err = ws.AdvanceImport(t.Context(), feed, "desk", "batch", spec, 0, 1); !errors.Is(err, revision.ErrConflict) || calls != 2 {
		t.Fatal(calls, err)
	}
	pages, err = ws.AdvanceImport(t.Context(), feed, "desk", "batch", spec, 1, 2)
	if err != nil || len(pages) != 1 || !pages[0].Complete || pages[0].Number != 2 || len(pages[0].Problems) != 1 || len(pages[0].Records) != 1 {
		t.Fatal(pages, err)
	}
	if p, err := ws.AdvanceImport(t.Context(), feed, "desk", "batch", spec, 2, 2); err != nil || len(p) != 0 || calls != 3 {
		t.Fatal(p, err, calls)
	}
	run, err := ws.Start(RunRequest{Description: "review", Factory: "desk", Import: &ImportPageRef{ID: "batch", Page: 1}})
	if err != nil || run.Records[0].ID != "9007199254740993" || run.Source.Revision != 1 {
		t.Fatal(run, err)
	}
	origin := pages[0].Lineage["2"]
	if origin.Pointer != "/rows/1" || origin.Feed != "feed" || origin.Import == nil || origin.Import.Page != 2 || origin.SHA256 == "" {
		t.Fatal(origin)
	}
	if other, err := ws.ImportPages("other", "batch"); err != nil || len(other) != 0 {
		t.Fatal(other, err)
	}
}
