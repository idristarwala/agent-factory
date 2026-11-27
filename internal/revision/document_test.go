package revision

import (
	"errors"
	"testing"
)

func TestConfigurationRevision(t *testing.T) {
	s := NewMemory()
	one, err := SaveDocument(s, "tool/lookup", 0, "human", []byte(`{"a":2,"b":1}`))
	if err != nil {
		t.Fatal(err)
	}
	same, err := SaveDocument(s, one.ID, 1, "human", []byte(`{ "b":1, "a":2 }`))
	if err != nil || same.Version != 1 {
		t.Fatal(same, err)
	}
	next, err := SaveDocument(s, one.ID, 1, "steward", []byte(`{"a":3,"b":1}`))
	if err != nil || next.Version != 2 {
		t.Fatal(next, err)
	}
	if _, err = SaveDocument(s, one.ID, 1, "human", []byte(`{}`)); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	h, err := Documents(s, one.ID)
	if err != nil || len(h) != 2 || string(h[0].Data) != string(one.Data) {
		t.Fatal(h, err)
	}
}
