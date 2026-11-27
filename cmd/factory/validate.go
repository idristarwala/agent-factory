package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/idristarwala/agent-factory/schemas"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

func validateFiles(paths []string) error {
	if len(paths) == 0 {
		for _, candidate := range []string{"factory.yaml", "pack.yaml", "pack.yml", "pack.json", "packs/pack.yaml", "packs/pack.json"} {
			if _, err := os.Stat(candidate); err == nil {
				paths = append(paths, candidate)
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		if len(paths) == 0 {
			if _, err := os.Stat("examples/starter/factory.yaml"); err == nil {
				paths = []string{"examples/starter/factory.yaml"}
			} else {
				return fmt.Errorf("supply a pack or factory.yaml path")
			}
		}
	}
	schema, err := jsonschema.CompileString("pack.schema.json", schemas.Pack)
	if err != nil {
		return err
	}
	for _, path := range paths {
		if filepath.Base(path) != "factory.yaml" && filepath.Base(path) != "factory.yml" {
			if err := validatePack(schema, path); err != nil {
				return err
			}
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var manifest struct{ Packs []struct{ Path string } }
		if err := yaml.Unmarshal(raw, &manifest); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if len(manifest.Packs) == 0 {
			return fmt.Errorf("%s: no pack paths declared", path)
		}
		for _, ref := range manifest.Packs {
			if strings.TrimSpace(ref.Path) == "" {
				return fmt.Errorf("%s: empty pack path", path)
			}
			candidate := ref.Path
			if !filepath.IsAbs(candidate) {
				candidate = filepath.Join(filepath.Dir(path), candidate)
			}
			if err := validatePack(schema, candidate); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePack(schema *jsonschema.Schema, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		for _, name := range []string{"pack.yaml", "pack.yml", "pack.json"} {
			file := filepath.Join(path, name)
			if entry, err := os.Stat(file); err == nil {
				if !entry.Mode().IsRegular() {
					return fmt.Errorf("%s: expected a pack file", file)
				}
				return validatePack(schema, file)
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		return fmt.Errorf("%s: no pack.yaml, pack.yml or pack.json", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var value any
	if filepath.Ext(path) == ".json" {
		err = json.Unmarshal(raw, &value)
	} else {
		decoder := yaml.NewDecoder(strings.NewReader(string(raw)))
		err = decoder.Decode(&value)
		if err == nil && decoder.Decode(new(any)) != io.EOF {
			return fmt.Errorf("%s: expected one YAML document", path)
		}
	}
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	// Convert YAML numbers and reject map keys that cannot be represented in JSON.
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err = json.Unmarshal(data, &value); err != nil {
		return err
	}
	if err = schema.Validate(value); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	fmt.Println("Validated pack", path)
	return nil
}
