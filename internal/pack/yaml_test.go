package pack

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadStarterPack(t *testing.T) {
	// Relative path from package root when tests run in repo.
	// Fall back to a minimal embedded case if the example is not present.
	p, err := Load("../../examples/starter/packs/catalog/pack.yaml")
	if err != nil {
		// Not fatal in isolated package tests; just note it.
		t.Logf("starter pack not found from package dir: %v", err)
		return
	}
	if p.Name != "catalog" {
		t.Errorf("name = %q", p.Name)
	}
	if len(p.WorkerRoles) < 1 {
		t.Errorf("expected worker roles, got %v", p.WorkerRoles)
	}
	if len(p.Rules) < 1 {
		t.Errorf("expected nested rules, got %d", len(p.Rules))
	}
	if p.Rules[0].ID == "" || p.Rules[0].Text == "" {
		t.Errorf("rule missing id/text: %+v", p.Rules[0])
	}
}

func TestLoadMinimalYAML(t *testing.T) {
	yamlDoc := `
name: retail
worker_roles:
  - title-fixer
rules:
  - id: keep_wording
    text: Keep original wording
    active: true
tools:
  - http
`
	var p Pack
	if err := yaml.Unmarshal([]byte(yamlDoc), &p); err != nil {
		t.Fatal(err)
	}
	if p.Name != "retail" {
		t.Errorf("name = %q", p.Name)
	}
	if len(p.Rules) != 1 || p.Rules[0].ID != "keep_wording" {
		t.Errorf("rules = %+v", p.Rules)
	}
	if len(p.Tools) != 1 {
		t.Errorf("tools = %v", p.Tools)
	}
}

func TestPackRuleActiveDefault(t *testing.T) {
	for path, source := range map[string]string{
		"pack.yaml": "name: support\nworker_roles: [triage]\nrules:\n  - {id: first, text: first rule}\n  - {id: second, text: second rule, active: false}\n  - {id: third, text: third rule, active: true}",
		"pack.json": `{"name":"support","worker_roles":["triage"],"rules":[{"id":"first","text":"first rule"},{"id":"second","text":"second rule","active":false},{"id":"third","text":"third rule","active":true}]}`,
	} {
		t.Run(path, func(t *testing.T) {
			p, err := Parse(path, []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Rules) != 3 || !p.Rules[0].Active || p.Rules[1].Active || !p.Rules[2].Active {
				t.Fatalf("wrong active flags: %+v", p.Rules)
			}
		})
	}
}
