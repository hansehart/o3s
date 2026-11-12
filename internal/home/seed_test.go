package home_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/home"
)

// A folder counts as a home once it holds the settings or the stack.
func TestExistsKnowsAHome(t *testing.T) {
	empty := t.TempDir()
	written := anInitializedHome(t)

	got := []bool{home.Exists(empty), home.Exists(written)}

	// Require the empty folder apart from the home
	if diff := cmp.Diff([]bool{false, true}, got); diff != "" {
		t.Errorf("Exists() (-want +got):\n%s", diff)
	}
}

// A home that lost its settings or never had its workspace gets them, and a whole home gets nothing.
func TestSeedWritesWhatTheHomeLacks(t *testing.T) {
	dir := anInitializedHome(t)
	// Stop when the settings stay in place
	if err := os.Remove(filepath.Join(dir, "o3s.json")); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	seeded, err := home.Seed(dir)
	// Stop when the seeding fails
	if err != nil {
		t.Fatalf("Seed() = %v", err)
	}
	again, err := home.Seed(dir)

	// Stop when the second seeding fails
	if err != nil {
		t.Fatalf("Seed() again = %v", err)
	}
	want := []string{filepath.Join(dir, "o3s.json"), filepath.Join(dir, filepath.Base(dir)+".code-workspace")}
	// Require both files seeded once
	if diff := cmp.Diff(want, seeded); diff != "" {
		t.Errorf("Seed() (-want +got):\n%s", diff)
	}
	// Require a whole home left as it is
	if len(again) != 0 {
		t.Errorf("Seed() again = %v, want nothing", again)
	}
}

// A release binary writes its own path as the start hook, so the editor finds it whatever its PATH holds.
func TestInitWritesTheBinaryAsTheStartHook(t *testing.T) {
	tests := []struct {
		version, executable, want string
	}{
		{"0.7.0", "/home/ubuntu/.local/bin/o3s", `"initializeCommand": ["/home/ubuntu/.local/bin/o3s", "init"]`},
		{"dev", "/tmp/go-build/o3s", `"initializeCommand": "o3s init"`},
	}
	// Check every kind of build
	for _, tt := range tests {
		home.Version, home.Executable = tt.version, tt.executable
		t.Cleanup(func() { home.Version, home.Executable = "dev", "" })
		dir := anInitializedHome(t)

		data, err := os.ReadFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"))

		// Report a start hook that differs
		if err != nil || !strings.Contains(string(data), tt.want) {
			t.Errorf("version %s: config lacks %s (%v)", tt.version, tt.want, err)
		}
	}
}

// A new home's workspace opens the home and the projects, where the work survives a rebuild.
func TestTheWorkspaceOpensTheProjects(t *testing.T) {
	dir := anInitializedHome(t)
	// Stop when the workspace stays unwritten
	if _, err := home.Seed(dir); err != nil {
		t.Fatalf("Setup: Seed() = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, filepath.Base(dir)+".code-workspace"))

	// Stop when the workspace stays unread
	if err != nil {
		t.Fatalf("ReadFile() = %v", err)
	}
	// Require both folders in the workspace
	for _, folder := range []string{`"path": "/home/ubuntu/o3s"`, `"path": "/home/ubuntu/projects"`} {
		// Report a folder the workspace leaves out
		if !strings.Contains(string(data), folder) {
			t.Errorf("workspace lacks %s:\n%s", folder, data)
		}
	}
}
