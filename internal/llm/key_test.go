package llm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadKeyFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.txt")
	os.WriteFile(path, []byte("secret-key\n"), 0600)

	key, err := LoadKey(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if key != "secret-key" {
		t.Errorf("got %q", key)
	}
}

func TestLoadKeyInline(t *testing.T) {
	key, err := LoadKey("", "inline-key")
	if err != nil {
		t.Fatal(err)
	}
	if key != "inline-key" {
		t.Errorf("got %q", key)
	}
}
