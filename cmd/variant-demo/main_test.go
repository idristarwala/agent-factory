package main

import (
	"bytes"
	"strings"
	"testing"
)

const inputs = `{"entities":[{"id":"1","attributes":{"group":"x","title":"Tool Red"}},{"id":"2","attributes":{"group":"x","title":"Tool Blue"}}]`

func TestIndependentLabels(t *testing.T) {
	var out bytes.Buffer
	for _, tc := range []struct{ labels, score string }{
		{`, "labels":[{"id":"1","group":"a"},{"id":"2","group":"a"}]`, "tp=1 fp=0 fn=0"},
		{`, "labels":[{"id":"1","group":"a"},{"id":"2","group":""}]`, "tp=0 fp=1 fn=0"},
	} {
		out.Reset()
		if err := benchmark([]byte(inputs+tc.labels+"}"), &out); err != nil {
			t.Fatal(err)
		}
		if strings.Count(out.String(), tc.score) != 3 {
			t.Fatal(out.String())
		}
	}
	out.Reset()
	if err := benchmark([]byte(inputs+"}"), &out); err != nil || strings.Contains(out.String(), "f1=") {
		t.Fatal(out.String(), err)
	}
	for _, l := range []string{`[]`, `[{"id":"1","group":"a"}]`, `[{"id":"3","group":"a"}]`, `[{"id":"1","group":"a"},{"id":"1","group":"a"}]`, `[{"id":"1"}]`} {
		if err := benchmark([]byte(inputs+`,"labels":`+l+"}"), &out); err == nil {
			t.Fatal(l)
		}
	}
}
