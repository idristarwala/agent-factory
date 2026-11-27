package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/idristarwala/agent-factory/internal/pack"
)

type packReceipt struct {
	Status  string   `json:"status"`
	Commit  string   `json:"commit"`
	PackSHA string   `json:"pack_sha256"`
	Tests   []string `json:"tests"`
}

func fixtureReceipt(path string, out io.Writer) error {
	dir := filepath.Dir(path)
	head, err := exec.Command("git", "-C", dir, "rev-parse", "--verify", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("resolve pack commit: %w", err)
	}
	commit := strings.TrimSpace(string(head))
	readFile := func(name string) ([]byte, error) {
		data, err := exec.Command("git", "-C", dir, "cat-file", "blob", commit+":./"+filepath.ToSlash(name)).Output()
		if err != nil {
			return nil, fmt.Errorf("read committed %s: %w", name, err)
		}
		return data, nil
	}
	data, err := readFile(filepath.Base(path))
	if err != nil {
		return err
	}
	p, err := pack.Parse(path, data)
	if err != nil {
		return err
	}
	files, err := exec.Command("git", "-C", dir, "ls-tree", "--full-tree", "-z", commit+":./fixtures").Output()
	if err != nil {
		return fmt.Errorf("list committed fixtures: %w", err)
	}
	receipt := packReceipt{Status: "passed", Commit: commit, PackSHA: fmt.Sprintf("%x", sha256.Sum256(data))}
	for _, entry := range strings.Split(string(files), "\x00") {
		metadata, name, ok := strings.Cut(entry, "\t")
		if !ok || strings.HasPrefix(metadata, "040000 tree ") || filepath.Ext(name) != ".json" {
			continue
		}
		file := filepath.Join("fixtures", name)
		if err := checkFixture(file, ".", p, readFile); err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		receipt.Tests = append(receipt.Tests, filepath.ToSlash(file))
	}
	if len(receipt.Tests) == 0 {
		return fmt.Errorf("%s: no committed JSON fixtures", path)
	}
	return json.NewEncoder(out).Encode(receipt)
}
