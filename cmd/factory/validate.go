package main

import (
	"errors"
	"fmt"

	"os"
	"path/filepath"
	"strings"

	"github.com/idristarwala/agent-factory/internal/pack"
	"github.com/idristarwala/agent-factory/internal/tools"

	"gopkg.in/yaml.v3"
)

func validateFiles(paths []string) error {
	if len(paths) > 0 && (paths[0] == "--rule" || paths[0] == "--tools") {
		if len(paths) == 1 {
			return fmt.Errorf("usage: factory validate %s FILE...", paths[0])
		}
		for _, path := range paths[1:] {
			if paths[0] == "--tools" {
				if err := tools.ValidateEvidenceFile(path); err != nil {
					return fmt.Errorf("%s: %w", path, err)
				}
				fmt.Printf("Validated evidence tools %s\n", path)
				continue
			}
			rule, err := pack.LoadRule(path)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			fmt.Printf("Validated rule %s (%s)\n", path, rule.ID)
		}
		return nil
	}
	return eachPack(paths, func(path string, p *pack.Pack) error {
		fmt.Printf("Validated pack %s (%s)\n", path, p.Name)
		return nil
	})
}

func eachPack(paths []string, visit func(string, *pack.Pack) error) error {
	if len(paths) == 0 {
		for _, candidate := range []string{"factory.yaml", "factory.yml", "pack.yaml", "pack.yml", "pack.json", "packs/pack.yaml", "packs/pack.json"} {
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
			if err := visitPack(path, visit); err != nil {
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
			if err := visitPack(candidate, visit); err != nil {
				return err
			}
		}
	}
	return nil
}

func visitPack(path string, visit func(string, *pack.Pack) error) error {
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
				return visitPack(file, visit)
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
	return visit(path, definition)
}
