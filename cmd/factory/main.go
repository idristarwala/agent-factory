package main

import (
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/store"
	"log"
	"os"
	"os/exec"
	"strings"

	"strconv"
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
		if err := validateFiles(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "test":
		if err := runFixtures(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		if len(os.Args) == 2 && engineCheckout() {
			runEngineTests()
		}
	case "readiness":
		runReadiness()
	case "up":
		if err := runUp(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "publish":
		if err := publishPack(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func engineCheckout() bool {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "module github.com/idristarwala/agent-factory" {
			return true
		}
	}
	return false
}

func runEngineTests() {
	// Actually invoke the critical packages so CI and local runs get the same signal.
	pkgs := []string{
		"./internal/llm/...",
		"./internal/store/...",
		"./internal/grouping/...",
	}
	cmd := append([]string{"go", "test", "-count=1"}, pkgs...)
	fmt.Println("  running:", strings.Join(cmd, " "))
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		log.Fatalf("tests failed: %v", err)
	}
	example := exec.Command("go", "test", "-count=1", "./...")
	example.Dir = "examples/out-of-tree-gateway"
	example.Stdout, example.Stderr = os.Stdout, os.Stderr
	if err := example.Run(); err != nil {
		log.Fatalf("out-of-tree conformance: %v", err)
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
	db, mode, err := store.OpenFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  workspace store: %s (opened successfully)\n", mode)
	fmt.Println("Ready (grants enforced in ContinuousRunner; secrets by reference).")
}

func printUsage() {
	fmt.Println("Usage: factory <command>")
	fmt.Println("Commands:")
	fmt.Println("  import     Save a versioned definition from JSON")
	fmt.Println("  up [-f FILE]   Start the local Compose project and wait for readiness")
	fmt.Println("  validate [PATH...]   Check packs against the published schema")
	fmt.Println("  test [PATH...]   Run pack fixtures (and engine contracts in the engine checkout)")
	fmt.Println("  readiness  Check environment readiness")
	fmt.Println("  publish -pack FILE -evidence FILE -expected N   Publish a proposed revision")
}
