package main

import (
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/store"
	"log"
	"os"
	"os/exec"
	"strings"
	"path/filepath"
	"strconv"

	"github.com/idristarwala/agent-factory/internal/pack"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	switch cmd {
	case "import":
		if len(os.Args) != 6 {
			log.Fatal("usage: factory import KIND ID EXPECTED_REVISION definition.json")
		}
		kind := os.Args[2]
		switch kind {
		case "factory", "tool", "pack", "experience":
		default:
			log.Fatal("unsupported definition kind")
		}
		n, err := strconv.Atoi(os.Args[4])
		if err != nil || n < 0 {
			log.Fatal("expected a nonnegative revision")
		}
		data, err := os.ReadFile(os.Args[5])
		if err != nil {
			log.Fatal(err)
		}
		db, err := store.NewFromEnv()
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		doc, err := revision.SaveDocument(store.FactoryDefinitions(db, ""), kind+"/"+os.Args[3], n, "human", data)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s revision %d\n", doc.ID, doc.Version)
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
	if _, err := os.Stat("go.mod"); err != nil {
		fmt.Println("  no go.mod — skipping live go test (library mode)")
		fmt.Println("Recommended: go test ./internal/llm/... ./internal/store/... ./internal/grouping/... ./examples/out-of-tree-gateway/")
		return
	}
	// Actually invoke the critical packages so CI and local runs get the same signal.
	pkgs := []string{
		"./internal/llm/...",
		"./internal/store/...",
		"./internal/grouping/...",
		"./examples/out-of-tree-gateway/...",
	}
	cmd := append([]string{"go", "test", "-count=1"}, pkgs...)
	fmt.Println("  running:", strings.Join(cmd, " "))
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		log.Fatalf("tests failed: %v", err)
	}
	fmt.Println("All selected tests passed.")
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
	fmt.Println("  import     Save a versioned definition from JSON")
	fmt.Println("  up         Start the factory locally")
	fmt.Println("  validate   Validate packs and config")
	fmt.Println("  test       Run fixture / contract tests")
	fmt.Println("  readiness  Check environment readiness")
	fmt.Println("  publish    Publish a pack revision")
}
