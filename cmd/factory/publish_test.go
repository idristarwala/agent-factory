package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/pack"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/store"
)

func TestPublishFromUserRepository(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("FACTORY_STORE_MODE", "")
	t.Setenv("FACTORY_DB_PATH", filepath.Join(dir, "data.sqlite"))
	t.Setenv("FACTORY_ID", "test-workspace")
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q")
	if err := os.Mkdir("support", 0700); err != nil {
		t.Fatal(err)
	}
	path := "support/pack.yaml"
	source := "name: support\nworker_roles: [triage]\n"
	write(path, source)
	git("add", path)
	git("-c", "user.name=CI", "-c", "user.email=ci@example.org", "commit", "-qm", "Pack")
	receipt := fmt.Sprintf(`{"status":"passed","tests":["fixture"],"commit":"%s","pack_sha256":"%x"}`, git("rev-parse", "HEAD"), sha256.Sum256([]byte(source)))
	args := []string{"-pack", path, "-evidence", "result.json", "-expected", "0"}
	run := func(ok bool) {
		t.Helper()
		if err := publishPack(args); (err == nil) != ok {
			t.Fatalf("want success=%v, got %v", ok, err)
		}
	}
	write("result.json", strings.Replace(receipt, "passed", "failed", 1))
	run(false)
	write("result.json", receipt)
	run(true)
	run(true)
	write(path, source+"# uncommitted\n")
	run(false)
	write(path, source)
	revised := strings.Replace(receipt, "fixture", "fixture and contract", 1)
	write("result.json", revised)
	run(false)
	args[len(args)-1] = "1"
	run(true)
	db, err := store.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	h, err := revision.Documents(store.FactoryDefinitions(db, ""), "pack-release/support")
	if err != nil || len(h) != 2 {
		t.Fatalf("history has %d revisions: %v", len(h), err)
	}
	for i, expected := range []string{receipt, revised} {
		var release pack.Release
		if err := json.Unmarshal(h[i].Data, &release); err != nil {
			t.Fatal(err)
		}
		if h[i].Version != i+1 || release.Status != "proposed" || string(release.Pack) != source || string(release.Evidence) != expected {
			t.Fatalf("unexpected revision %d", i+1)
		}
	}
}
