package tools

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// readEvidenceFile accepts standalone evidence files and package pins in factory.yaml.
func readEvidenceFile(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil || (filepath.Base(path) != "factory.yaml" && filepath.Base(path) != "factory.yml") {
		return raw, err
	}
	var manifest struct {
		ToolPackages yaml.Node            `yaml:"tool_packages"`
		Evidence     map[string]any       `yaml:"evidence"`
		Other        map[string]yaml.Node `yaml:",inline"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&manifest); err != nil {
		return nil, err
	}
	if manifest.ToolPackages.Kind == 0 {
		return raw, nil
	}
	for _, key := range []string{"tools", "context", "grants"} {
		if _, present := manifest.Other[key]; present {
			return nil, fmt.Errorf("unexpected top-level %q alongside tool_packages; use packaged definitions and evidence.context or evidence.grants", key)
		}
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("expected one factory manifest")
	}
	var pins []struct{ Path, SHA256 string }
	if err := manifest.ToolPackages.Decode(&pins); err != nil || len(pins) == 0 {
		return nil, fmt.Errorf("tool_packages requires at least one path and sha256 pin")
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	definitions := map[string]any{}
	for _, pin := range pins {
		if !filepath.IsLocal(pin.Path) || len(pin.SHA256) != sha256.Size*2 {
			return nil, fmt.Errorf("tool package requires a relative path and a full SHA-256 digest")
		}
		data, err := root.ReadFile(pin.Path)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != pin.SHA256 {
			return nil, fmt.Errorf("tool package %q SHA-256 mismatch", pin.Path)
		}
		var pkg struct{ Tools map[string]any }
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&pkg); err != nil {
			return nil, fmt.Errorf("tool package %q: %w", pin.Path, err)
		}
		if len(pkg.Tools) == 0 || decoder.Decode(new(any)) != io.EOF {
			return nil, fmt.Errorf("tool package %q must contain one document with tool definitions", pin.Path)
		}
		for name, def := range pkg.Tools {
			if _, duplicate := definitions[name]; duplicate {
				return nil, fmt.Errorf("duplicate packaged tool %q", name)
			}
			definitions[name] = def
		}
	}
	if manifest.Evidence == nil {
		manifest.Evidence = make(map[string]any)
	}
	for name := range manifest.Evidence {
		if name != "context" && name != "grants" {
			return nil, fmt.Errorf("unexpected evidence setting %q in factory manifest", name)
		}
	}
	manifest.Evidence["tools"] = definitions
	return yaml.Marshal(manifest.Evidence)
}
