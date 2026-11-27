package main

import (
	"errors"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/store"
	"github.com/idristarwala/agent-factory/internal/tools"
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
	case "tools":
		if err := importTools(os.Args[2:], os.Stdout, os.Stderr); err != nil {
			log.Fatal(err)
		}
	case "validate":
		if err := validateFiles(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "test":
		if len(os.Args) >= 3 && os.Args[2] == "--receipt" {
			if len(os.Args) != 4 {
				log.Fatal("usage: factory test --receipt PACK_FILE")
			}
			if err := fixtureReceipt(os.Args[3], os.Stdout); err != nil {
				log.Fatal(err)
			}
			break
		}
		if err := runFixtures(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		if len(os.Args) == 2 && engineCheckout() {
			runEngineTests()
		}
	case "readiness":
		if err := runReadiness(); err != nil {
			log.Fatal(err)
		}
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

func runReadiness() error {
	var problems []error
	db, mode, err := store.OpenFromEnv()
	if err != nil {
		problems = append(problems, err)
	} else {
		if err = db.Close(); err != nil {
			problems = append(problems, err)
		} else {
			fmt.Printf("workspace store: %s (opened successfully)\n", mode)
		}
	}
	cfg, err := tools.LoadContinuousEvidence(os.Getenv("FACTORY_EVIDENCE_CONFIG"), os.Getenv("FACTORY_PACK"))
	if err != nil {
		problems = append(problems, errors.New("check FACTORY_EVIDENCE_CONFIG and FACTORY_PACK: the runner requires a valid recipe, its tool grants and a read-only propose grant"))
	} else if cfg != nil {
		if err := cfg.CheckCredentials(); err != nil {
			problems = append(problems, err)
		} else {
			fmt.Println("continuous evidence: recipe, grants and credential files ready")
		}
	}
	if path := os.Getenv("FACTORY_SOURCE_CONFIG"); path != "" {
		sampler, err := tools.LoadSourceSampler(path)
		if err != nil {
			problems = append(problems, errors.New("check FACTORY_SOURCE_CONFIG: expected valid tool configuration and fixed sampling URLs"))
		} else if err := sampler.CheckCredentials(); err != nil {
			problems = append(problems, err)
		} else {
			fmt.Println("API sources: configuration and credential files ready")
		}
	}
	return errors.Join(problems...)
}

func printUsage() {
	fmt.Println("Usage: factory <command>")
	fmt.Println("Commands:")
	fmt.Println("  import     Save a versioned definition from JSON")
	fmt.Println("  up [-f FILE]   Start the local Compose project and wait for readiness")
	fmt.Println("  validate [PATH...]   Check packs against the published schema")
	fmt.Println("  validate --tools FILE...   Check draft HTTP and MCP tool files")
	fmt.Println("  validate --rule FILE...   Check standalone rules against the published schema")
	fmt.Println("  test [PATH...]   Run pack fixtures (and engine contracts in the engine checkout)")
	fmt.Println("  test --receipt PACK_FILE   Test committed fixtures and emit publish evidence as JSON")
	fmt.Println("  readiness  Check environment readiness")
	fmt.Println("  tools import [flags] FILE   Generate reviewable GET evidence tools from OpenAPI")
	fmt.Println("  publish -pack FILE -evidence FILE -expected N   Publish a proposed revision")
}
