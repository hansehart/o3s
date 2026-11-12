package rebuild_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hansehart/o3s/internal/rebuild"
)

// aCLI writes a fake devcontainer CLI into a folder, recording how it is called.
func aCLI(t *testing.T, dir string) string {
	t.Helper()
	// Stop when the folder fails to appear
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	log := filepath.Join(dir, "calls.log")
	cli := "#!/bin/sh\necho \"$*\" >> " + log + "\necho rebuilt\n"
	// Stop when the fake stays unwritten
	if err := os.WriteFile(filepath.Join(dir, "devcontainer"), []byte(cli), 0o755); err != nil { //nolint:gosec // the test runs its fake
		t.Fatalf("Setup: %v", err)
	}
	return log
}

// The CLI o3s installed in its data folder rebuilds the home's cage the way the editor's rebuild does.
func TestRunRebuildsTheCage(t *testing.T) {
	data := t.TempDir()
	log := aCLI(t, filepath.Join(data, "o3s", "devcontainers", "bin"))
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("PATH", t.TempDir())
	cli, ok := rebuild.CLI()
	// Stop when the CLI stays unfound
	if !ok {
		t.Fatalf("CLI() = false, want the installed CLI")
	}
	var out strings.Builder

	err := rebuild.Run(t.Context(), cli, "/home/ubuntu/o3s", &out)

	// Stop when the rebuild fails
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	calls, _ := os.ReadFile(log)
	// Require the rebuild the editor runs
	if got, want := strings.TrimSpace(string(calls)), "up --workspace-folder /home/ubuntu/o3s --remove-existing-container"; got != want {
		t.Errorf("devcontainer called with %q, want %q", got, want)
	}
	// Require the CLI's output passed on
	if !strings.Contains(out.String(), "rebuilt") {
		t.Errorf("Run() output = %q, want the CLI's", out.String())
	}
}

// A CLI on PATH serves where o3s's data folder holds none, and a host without either reports none.
func TestCLIFindsTheCLIOnPath(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	bin := t.TempDir()
	aCLI(t, bin)
	t.Setenv("PATH", bin)

	_, onPath := rebuild.CLI()
	t.Setenv("PATH", t.TempDir())
	_, without := rebuild.CLI()

	// Require the CLI on PATH found, and none reported without it
	if !onPath || without {
		t.Errorf("CLI() on PATH = %v and without = %v, want true and false", onPath, without)
	}
}
