package pack
import "encoding/json"

import (
	"testing"
)

func TestSimpleYAMLToJSON(t *testing.T) {
	yaml := `
name: retail
worker_roles:
  - title-fixer
  - attribute-normalizer
rules:
  - keep original wording when possible
tools_allowed:
  - http
`
	js, err := SimpleYAMLToJSON([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	var p Pack
	if err := json.Unmarshal(js, &p); err != nil {
		t.Fatal(err)
	}
	if p.Name != "retail" {
		t.Errorf("name = %q", p.Name)
	}
	if len(p.WorkerRoles) != 2 {
		t.Errorf("roles = %v", p.WorkerRoles)
	}
}
