package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestUpUsesSelectedComposeProject(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("PATH", dir)
	t.Setenv("COMPOSE_FILE", "")
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(name, []byte(content), 0700); err != nil {
			t.Fatal(err)
		}
	}
	write("docker", "#!/bin/sh\nprintf '%s\\n' \"$@\" > args\nexit \"${DOCKER_TEST_STATUS:-0}\"\n")
	if err := os.Mkdir("deploy", 0700); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join("deploy", "compose.yaml"), "")
	verify := func(args []string, want string) {
		t.Helper()
		if err := runUp(args); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile("args")
		if err != nil || string(data) != want+"up\n--detach\n--wait\n" {
			t.Fatalf("compose arguments: %q (%v)", data, err)
		}
	}
	verify(nil, "compose\n-f\ndeploy/compose.yaml\n")
	t.Setenv("COMPOSE_FILE", "custom.yaml")
	verify(nil, "compose\n")
	verify([]string{"-f", "chosen file.yml"}, "compose\n-f\nchosen file.yml\n")
	t.Setenv("COMPOSE_FILE", "")
	write("compose.yml", "")
	verify(nil, "compose\n")
	t.Setenv("DOCKER_TEST_STATUS", "7")
	if err := runUp(nil); err == nil {
		t.Fatal("expected Compose failure")
	}
	if err := runUp([]string{"unexpected"}); err == nil {
		t.Fatal("expected invalid argument error")
	}
}
