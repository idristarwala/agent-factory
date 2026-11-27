package factory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"
	"strings"
	"testing"
)

func TestReviewedRecordLineage(t *testing.T) {
	s := revision.NewMemory()
	ws := Workspace{Store: s, Gateway: scripted{}}
	for key, data := range map[string]string{"factory/desk": `{"record_type":"case","fields":{"priority":"integer"}}`, "source-mapping/m": `{"record_type":"case","collection":"/rows","identity":["/id"],"fields":{"priority":"/priority"}}`} {
		if _, err := revision.SaveDocument(s, key, 0, "human", []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	const raw = `{"rows":[{"priority":0},{"priority":1,"id":"C","unmapped":"private"}]}`
	var sample any
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&sample); err != nil {
		t.Fatal(err)
	}
	receipt, err := SaveSampleReceipt(s, "desk", "inbox", sample)
	if err != nil {
		t.Fatal(err)
	}
	req := RunRequest{Factory: "desk", Description: "review", Source: &SourceInput{SourceRef: SourceRef{Mapping: "m", Revision: 1}, AllowPartial: true, Receipt: receipt, Payload: json.RawMessage(raw)}}
	r, err := ws.Start(req)
	if err != nil {
		t.Fatal(err)
	}
	if err = ws.Execute(r.ID); err != nil {
		t.Fatal(err)
	}
	r, err = ws.Get(r.ID)
	if err != nil || len(r.Decisions) != 1 {
		t.Fatal(r, err)
	}
	origin := r.Decisions[0].Lineage
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(`{"id":"C","priority":1,"unmapped":"private"}`)))
	if origin == nil || origin.Pointer != "/rows/1" || origin.SHA256 != digest || origin.Feed != "inbox" || origin.Receipt != receipt || origin.Revision != 1 {
		t.Fatal(origin)
	}
	if !strings.Contains(strings.Join(r.Decisions[0].Proposal.Evidence, " "), digest) {
		t.Fatal("missing proposal lineage")
	}
	req.Source.Payload = json.RawMessage(strings.Replace(raw, "private", "altered", 1))
	if _, err = ws.Start(req); err == nil {
		t.Fatal("accepted altered receipt payload")
	}
	if _, err = ws.verifiedFeed("other", receipt, sample); err == nil {
		t.Fatal("accepted cross-factory receipt")
	}
}
