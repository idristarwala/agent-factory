package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateManifestPacks(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "factory.yaml")
	if err := os.WriteFile(manifest, []byte("packs:\n  - path: ./packs\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "packs"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "packs", "pack.yaml")
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{"", false},
		{"name: valid\nworker_roles: []", false},
		{"name: valid\nworker_roles: [specialist]\nrules: [{id: only_id}]", false},
		{"name: valid\nworker_roles: [specialist]\n---\nname: another", false},
		{"name: valid\nworker_roles: [specialist]\nrules: [{id: grounded, text: Cite source}]", true},
	} {
		if err := os.WriteFile(path, []byte(tc.body), 0600); err != nil {
			t.Fatal(err)
		}
		if err := validateFiles([]string{manifest}); (err == nil) != tc.valid {
			t.Fatal(tc, err)
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := validateFiles([]string{manifest}); err == nil {
		t.Fatal("missing pack passed")
	}
}

func TestRuleFileAndPackShareContract(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		rule  string
		valid bool
	}{
		{`{"id":"keep","text":"Preserve source"}`, true},
		{`{"id":"keep","text":"Preserve source","active":false}`, true},
		{`{"id":"keep","active":true}`, false},
		{`{"id":"keep","text":"Preserve source","active":"no"}`, false},
	} {
		for _, ext := range []string{".yaml", ".json"} {
			ruleFile := filepath.Join(dir, "routing"+ext)
			packFile := filepath.Join(dir, "pack"+ext)
			if err := os.WriteFile(ruleFile, []byte(tc.rule), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(packFile, []byte(`{"name":"desk","worker_roles":["worker"],"rules":[`+tc.rule+`]}`), 0600); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{{"--rule", ruleFile}, {packFile}} {
				if err := validateFiles(args); (err == nil) != tc.valid {
					t.Fatalf("%v: %v", args, err)
				}
			}
		}
	}
	if err := validateFiles([]string{"--rule"}); err == nil {
		t.Fatal("expected missing file error")
	}
}
