package main

import (
	"fmt"
	"os"
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
		fmt.Println("All checks passed (skeleton).")
	case "test":
		fmt.Println("Running fixture tests with fake LLM gateway...")
		fmt.Println("All tests passed (skeleton).")
	case "readiness":
		fmt.Println("Checking tools, credentials, grants, and connectivity...")
		fmt.Println("Ready (skeleton).")
	case "up":
		fmt.Println("Starting factory (docker-compose recommended for full stack).")
		fmt.Println("Use: docker-compose up")
	case "publish":
		fmt.Println("Publishing pack revision as proposed release...")
		fmt.Println("Proposed (skeleton).")
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
