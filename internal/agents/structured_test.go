package agents

import "testing"

func TestValidateJSON(t *testing.T) {
	ok, err := ValidateJSON(`{"title": "ok", "desc": "fine"}`)
	if err != nil || ok["title"] != "ok" {
		t.Fatalf("expected valid: %v %v", ok, err)
	}

	fenced, err := ValidateJSON("```json\n{\"a\": 1}\n```")
	if err != nil || fenced["a"] == nil {
		t.Fatalf("fence strip failed: %v", err)
	}

	_, err = ValidateJSON("not json")
	if err == nil {
		t.Fatal("expected error")
	}
}
