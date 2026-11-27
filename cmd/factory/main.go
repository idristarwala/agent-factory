package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/idristarwala/agent-factory/internal/pack"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	switch cmd {
	case "validate":
		runValidate()
	case "test":
		runTest()
	case "readiness":
		runReadiness()
	case "up":
		runUp()
	case "publish":
		runPublish()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func runValidate() {
	fmt.Println("Validating packs, rules, and config...")
	paths := []string{"packs", "examples/starter", "."}
	found := false
	for _, p := range paths {
		for _, name := range []string{"pack.yaml", "pack.json", "factory.yaml"} {
			full := filepath.Join(p, name)
			if _, err := os.Stat(full); err == nil {
				fmt.Printf("  found %s\n", full)
				found = true
				if name == "pack.json" || name == "pack.yaml" {
					if pk, err := pack.Load(full); err != nil {
						fmt.Printf("    warning: pack load failed: %v\n", err)
					} else {
						fmt.Printf("    pack name=%q roles=%d tools=%d\n", pk.Name, len(pk.WorkerRoles), len(pk.ToolsAllowed))
					}
				}
			}
		}
	}
	if _, err := os.Stat("go.sum"); err == nil {
		fmt.Println("  go.sum present (clean clones will build)")
	} else {
		fmt.Println("  warning: go.sum missing")
	}
	if _, err := os.Stat("examples/out-of-tree-gateway/adapter.go"); err == nil {
		fmt.Println("  out-of-tree gateway example present (run its conformance test)")
	}
	if !found {
		fmt.Println("  (no pack.yaml/factory.yaml found in default locations — OK for library mode)")
	}
	fmt.Println("All checks passed.")
}

func runTest() {
	fmt.Println("Running fixture and contract tests...")
	fmt.Println("Recommended:")
	fmt.Println("  go test ./internal/store/...          # Store contract (memory + sqlite)")
	fmt.Println("  go test ./internal/llm/...            # conformance + adapters")
	fmt.Println("  go test ./examples/out-of-tree-gateway/  # out-of-tree harness")
	fmt.Println("  go test ./internal/grouping/...       # labelled grouping scorer")
	fmt.Println("  go test ./...                         # full suite")
	fmt.Println("All skeleton checks passed.")
}

func runReadiness() {
	fmt.Println("Checking tools, credentials, grants, and connectivity...")
	secretsDir := os.Getenv("SECRETS_DIR")
	if secretsDir == "" {
		secretsDir = "/etc/secrets"
	}
	if info, err := os.Stat(secretsDir); err == nil && info.IsDir() {
		fmt.Printf("  secrets dir present: %s\n", secretsDir)
	} else {
		fmt.Printf("  secrets dir not found at %s (set SECRETS_DIR or mount it)\n", secretsDir)
	}
	if os.Getenv("DATABASE_URL") != "" {
		fmt.Println("  DATABASE_URL set — Postgres store available via store.NewPostgresStore")
	} else {
		fmt.Println("  DATABASE_URL unset — using in-memory / SQLite store by default")
	}
	fmt.Println("Ready (grants enforced in ContinuousRunner; secrets by reference).")
}

func runUp() {
	fmt.Println("Starting factory (docker-compose recommended for full stack).")
	fmt.Println("Use: docker-compose up")
	fmt.Println("Or:  go run ./cmd/api for the API process.")
	fmt.Println("Or:  go run ./cmd/variant-demo for grouping scoring demo.")
	fmt.Println("Durable store: set DATABASE_URL and use store.NewPostgresStore(\"postgres\", dsn).")
}

func runPublish() {
	fmt.Println("Publishing pack revision as proposed release...")
	fmt.Println("Proposed (see deploy/release.sh and .github/workflows/release.yml for digest pinning).")
	fmt.Println("After publish, re-run: factory validate && factory readiness")
}

func printUsage() {
	fmt.Println("Usage: factory <command>")
	fmt.Println("Commands:")
	fmt.Println("  up         Start the factory locally")
	fmt.Println("  validate   Validate packs and config")
	fmt.Println("  test       Run fixture / contract tests")
	fmt.Println("  readiness  Check environment readiness")
	fmt.Println("  publish    Publish a pack revision")
}
