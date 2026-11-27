package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func runUp(args []string) error {
	flags := flag.NewFlagSet("up", flag.ContinueOnError)
	file := flags.String("f", "", "Compose file (default: discover the local project)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: factory up [-f FILE]")
	}
	// Preserve Compose's native root-file discovery, including override files.
	if *file == "" && os.Getenv("COMPOSE_FILE") == "" {
		for _, candidate := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml", "deploy/compose.yaml"} {
			if _, err := os.Stat(candidate); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return err
			}
			if candidate == "deploy/compose.yaml" {
				*file = candidate
			}
			break
		}
	}
	command := []string{"compose"}
	if *file != "" {
		command = append(command, "-f", *file)
	}
	command = append(command, "up", "--detach", "--wait")
	cmd := exec.Command("docker", command...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start Compose project: %w", err)
	}
	return nil
}
