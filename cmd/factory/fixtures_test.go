package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
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

func TestReceiptUsesOneCommittedSnapshot(t *testing.T) {
	dir := t.TempDir()
	if err := os.CopyFS(filepath.Join(dir, "pack"), os.DirFS("../../examples/service-desk")); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("%s: %v", out, err)
		}
	}
	git("init", "-q")
	git("add", "pack")
	commit := func() { git("-c", "user.name=CI", "-c", "user.email=ci@example.org", "commit", "-qm", "snapshot") }
	commit()
	if err := os.WriteFile("pack/factory.json", []byte("broken definition"), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := fixtureReceipt("pack/pack.yaml", &out); err != nil {
		t.Fatal(err)
	}
	var receipt packReceipt
	if err := json.Unmarshal(out.Bytes(), &receipt); err != nil || receipt.Status != "passed" || len(receipt.Tests) != 1 || receipt.Tests[0] != "fixtures/routing.json" {
		t.Fatal(receipt, err)
	}
	t.Setenv("FACTORY_STORE_MODE", "sqlite")
	t.Setenv("FACTORY_DB_PATH", filepath.Join(dir, "store.sqlite"))
	if err := os.WriteFile("receipt.json", out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publishPack([]string{"-pack", "pack/pack.yaml", "-evidence", "receipt.json", "-expected", "0"}); err != nil {
		t.Fatal(err)
	}
	git("add", "pack")
	commit()
	out.Reset()
	if err := fixtureReceipt("pack/pack.yaml", &out); err == nil || out.Len() != 0 {
		t.Fatal(out.String(), err)
	}
}
