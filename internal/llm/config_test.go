package llm

import (
	"testing"
)

func TestApplyExtraHeaders(t *testing.T) {
	base := map[string]string{"Authorization": "Bearer x"}
	extra := map[string]string{"X-Consumer-Id": "my-app"}
	out := ApplyExtraHeaders(base, extra)
	if out["Authorization"] != "Bearer x" {
		t.Error("base header lost")
	}
	if out["X-Consumer-Id"] != "my-app" {
		t.Error("extra header missing")
	}
}
