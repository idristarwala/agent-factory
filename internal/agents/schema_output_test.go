package agents

import "testing"

func TestBuildResponseSchema(t *testing.T) {
	allowed := []string{"title", "description"}
	required := []string{"title"}
	s := BuildResponseSchema(allowed, required)
	if s["type"] != "object" {
		t.Fatal("expected object")
	}
	if s["additionalProperties"] != false {
		t.Fatal("expected additionalProperties false")
	}
	props, ok := s["properties"].(map[string]any)
	if !ok || len(props) != 2 {
		t.Fatal("expected two properties")
	}
	req, ok := s["required"].([]string)
	if !ok || len(req) != 1 || req[0] != "title" {
		t.Fatal("expected required title")
	}
}
