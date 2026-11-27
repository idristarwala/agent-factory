package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/idristarwala/agent-factory/internal/pack"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/store"
)

func publishPack(args []string) error {
	flags := flag.NewFlagSet("publish", flag.ContinueOnError)
	path := flags.String("pack", "", "tracked YAML or JSON pack file")
	evidence := flags.String("evidence", "", "passing CI evidence JSON file")
	expected := flags.Int("expected", -1, "current release revision, zero for the first proposal")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *expected < 0 || *path == "" || *evidence == "" || flags.NArg() != 0 {
		return fmt.Errorf("usage: factory publish -pack FILE -evidence FILE -expected N")
	}
	if os.Getenv("FACTORY_STORE_MODE") == "memory" {
		return fmt.Errorf("publish requires a persistent store (SQLite or PostgreSQL)")
	}
	data, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	p, err := pack.Parse(*path, data)
	if err != nil {
		return err
	}
	if strings.TrimSpace(p.Name) == "" || strings.ContainsAny(p.Name, "/\\") || p.Name == "." || p.Name == ".." {
		return fmt.Errorf("pack name must be a single release ID")
	}
	dir := filepath.Dir(*path)
	head, err := exec.Command("git", "-C", dir, "rev-parse", "--verify", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("resolve pack commit: %w", err)
	}
	commit := strings.TrimSpace(string(head))
	blob, err := exec.Command("git", "-C", dir, "cat-file", "blob", commit+":./"+filepath.Base(*path)).Output()
	if err != nil || !bytes.Equal(blob, data) {
		return fmt.Errorf("pack must match the file committed at %s", commit)
	}
	raw, err := os.ReadFile(*evidence)
	if err != nil {
		return err
	}
	var receipt packReceipt
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return fmt.Errorf("decode evidence: %w", err)
	}
	if receipt.Status != "passed" || receipt.Commit != commit || receipt.PackSHA != digest || len(receipt.Tests) == 0 {
		return fmt.Errorf("evidence must report passing tests for this commit and pack_sha256")
	}
	for _, name := range receipt.Tests {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("evidence contains an unnamed test")
		}
	}
	release := pack.Release{Status: "proposed", Commit: commit, PackSHA: digest, EvidenceSHA: fmt.Sprintf("%x", sha256.Sum256(raw)), Pack: data, Evidence: raw}
	db, err := store.NewFromEnv()
	if err != nil {
		return err
	}
	defer db.Close()
	saved, err := saveRelease(store.FactoryDefinitions(db, ""), p.Name, *expected, release)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(saved)
}

func saveRelease(s revision.Store, name string, expected int, release pack.Release) (revision.Document, error) {
	id := "pack-release/" + name
	data, err := json.Marshal(release)
	if err != nil {
		return revision.Document{}, err
	}
	saved, err := revision.SaveDocument(s, id, expected, "publisher", data)
	if !errors.Is(err, revision.ErrConflict) {
		return saved, err
	}
	// A caller may have lost the response after the original insert committed.
	history, readErr := revision.Documents(s, id)
	if readErr != nil {
		return revision.Document{}, readErr
	}
	if len(history) == expected+1 && len(history) > 0 {
		latest := history[len(history)-1]
		var previous pack.Release
		if err := json.Unmarshal(latest.Data, &previous); err != nil {
			return revision.Document{}, err
		}
		if previous.Status == release.Status && previous.Commit == release.Commit && previous.PackSHA == release.PackSHA && previous.EvidenceSHA == release.EvidenceSHA {
			return latest, nil
		}
	}
	return revision.Document{}, err
}
