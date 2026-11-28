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

func TestValidateJSONRecoversCommonModelMistakes(t *testing.T) {
	// Trailing comma
	m, err := ValidateJSON(`{"title": "Widget", "desc": "Nice",}`)
	if err != nil || m["title"] != "Widget" {
		t.Fatalf("trailing comma not recovered: %v %v", m, err)
	}

	// Surrounding prose
	m, err = ValidateJSON(`Here is the update: {"title": "Fixed Widget"} Thanks!`)
	if err != nil || m["title"] != "Fixed Widget" {
		t.Fatalf("prose extraction failed: %v %v", m, err)
	}

	// Fence + trailing comma + prose
	m, err = ValidateJSON("```json\nThe result is {\"title\": \"OK\",}\n```")
	if err != nil || m["title"] != "OK" {
		t.Fatalf("combined recovery failed: %v %v", m, err)
	}

	// Line and block comments
	m, err = ValidateJSON(`{"title": "Widget" // keep this
	, "desc": "Nice" /* ignore me */}`)
	if err != nil || m["title"] != "Widget" || m["desc"] != "Nice" {
		t.Fatalf("comment stripping failed: %v %v", m, err)
	}

	// Single-quoted strings (JS-style)
	m, err = ValidateJSON(`{'title': 'Widget', 'desc': 'Nice'}`)
	if err != nil || m["title"] != "Widget" || m["desc"] != "Nice" {
		t.Fatalf("single-quote recovery failed: %v %v", m, err)
	}

	// Unquoted keys
	m, err = ValidateJSON(`{title: "Widget", desc: "Nice"}`)
	if err != nil || m["title"] != "Widget" || m["desc"] != "Nice" {
		t.Fatalf("unquoted-key recovery failed: %v %v", m, err)
	}
}
