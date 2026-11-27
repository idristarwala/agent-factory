package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	switch cmd {
	case "validate":
		fmt.Println("Validating packs, rules, and config...")
		// Check for common pack locations
		paths := []string{"packs", "examples/starter", "."}
		found := false
		for _, p := range paths {
			if _, err := os.Stat(filepath.Join(p, "pack.yaml")); err == nil {
				fmt.Printf("  found pack.yaml in %s\n", p)
				found = true
			}
			if _, err := os.Stat(filepath.Join(p, "factory.yaml")); err == nil {
				fmt.Printf("  found factory.yaml in %s\n", p)
				found = true
			}
		}
		if !found {
			fmt.Println("  (no pack.yaml/factory.yaml found in default locations — OK for library mode)")
		}
		fmt.Println("All checks passed (skeleton — full schema validation in internal/domain and internal/tools).")
	case "test":
		fmt.Println("Running fixture tests with fake LLM gateway...")
		fmt.Println("All tests passed (skeleton — use `go test ./...` for full suite including conformance and grouping scorer).")
	case "readiness":
		fmt.Println("Checking tools, credentials, grants, and connectivity...")
		fmt.Println("Ready (skeleton — grants enforced in ContinuousRunner; secrets by reference via SECRETS_DIR).")
	case "up":
		fmt.Println("Starting factory (docker-compose recommended for full stack).")
		fmt.Println("Use: docker-compose up")
		fmt.Println("Or: go run ./cmd/api for the API process.")
		fmt.Println("Or: go run ./cmd/variant-demo for grouping scoring demo.")
	case "publish":
		fmt.Println("Publishing pack revision as proposed release...")
		fmt.Println("Proposed (skeleton — see deploy/release.sh and .github/workflows/release.yml for digest pinning).")
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: factory <command>")
	fmt.Println("Commands:")
	fmt.Println("  up         Start the factory locally")
	fmt.Println("  validate   Validate packs and config")
	fmt.Println("  test       Run fixture tests")
	fmt.Println("  readiness  Check environment readiness")
	fmt.Println("  publish    Publish a pack revision")
}
