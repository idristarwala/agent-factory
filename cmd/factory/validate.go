package main

import (
	"errors"
	"fmt"

	"os"
	"path/filepath"
	"strings"

	"github.com/idristarwala/agent-factory/internal/pack"

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
	for _, path := range paths {
		if filepath.Base(path) != "factory.yaml" && filepath.Base(path) != "factory.yml" {
			if err := validatePack(path); err != nil {
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
			if err := validatePack(candidate); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePack(path string) error {
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
				return validatePack(file)
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		return fmt.Errorf("%s: no pack.yaml, pack.yml or pack.json", path)
	}
	definition, err := pack.Load(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	fmt.Printf("Validated pack %s (%s)\n", path, definition.Name)
	return nil
}
