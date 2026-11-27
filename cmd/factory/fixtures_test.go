package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStandalonePackFixtures(t *testing.T) {
	source := "../../examples/service-desk"
	dir := t.TempDir()
	target := filepath.Join(dir, "user-pack")
	if err := os.CopyFS(target, os.DirFS(source)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "factory.yaml"), []byte("packs:\n - path: user-pack\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("FACTORY_STORE_MODE", "unsupported")
	if err := runFixtures(nil); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(target, "fixtures", "routing.json")
	original, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		old, new string
		pass     bool
	}{
		{`"expected_patch":{"assigned_team":"Workplace IT"}`, `"expected_patch":{"assigned_team":"Other"}`, false},
		{`"expected_approved":true`, `"expected_approved":false`, false},
		{`"expected_approved":true`, `"unknown":true`, false},
	} {
		if err := os.WriteFile(file, []byte(strings.Replace(string(original), tc.old, tc.new, 1)), 0600); err != nil {
			t.Fatal(err)
		}
		if err := runFixtures(nil); (err == nil) != tc.pass {
			t.Fatalf("fixture change %s: %v", tc.new, err)
		}
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := runFixtures(nil); err == nil {
		t.Fatal("no fixtures reported success")
	}
}
