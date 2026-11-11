package home_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/home"
)

// paths lists where each change writes.
func paths(changes []home.FileChange) []string {
	var out []string
	// Take every change's path, marked new where the home lacks the file
	for _, c := range changes {
		mark := "~ "
		// Mark a file the home lacks
		if c.New {
			mark = "+ "
		}
		out = append(out, mark+c.Path)
	}
	return out
}

// A home o3s has just written holds the stack the binary ships.
func TestRefreshFindsNothingInANewHome(t *testing.T) {
	dir := anInitializedHome(t)

	got, err := home.Refresh(dir)

	// Require a new home to match the binary
	if err != nil || len(got) != 0 {
		t.Errorf("Refresh() = %v, %v, want no change", paths(got), err)
	}
}

// A generated file the user edited or lost comes back from the binary.
func TestRefreshBringsBackEditedAndLostFiles(t *testing.T) {
	dir := anInitializedHome(t)
	compose := filepath.Join(dir, ".devcontainer", "compose.yaml")
	// Stop when the edit fails
	if err := os.WriteFile(compose, []byte("services: {}\n"), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Stop when the schema stays in place
	if err := os.Remove(filepath.Join(dir, ".devcontainer", "o3s.schema.json")); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	got, err := home.Refresh(dir)

	// Stop when the refresh fails
	if err != nil {
		t.Fatalf("Refresh() = %v", err)
	}
	// Require both files named with what happens to them
	if diff := cmp.Diff([]string{"~ compose.yaml", "+ o3s.schema.json"}, paths(got)); diff != "" {
		t.Errorf("Refresh() (-want +got):\n%s", diff)
	}
	// Stop when the files fail to write
	if err := home.Write(dir, got); err != nil {
		t.Fatalf("Write() = %v", err)
	}
	again, err := home.Refresh(dir)
	// Require the home to match the binary once written
	if err != nil || len(again) != 0 {
		t.Errorf("Refresh() after Write = %v, %v, want no change", paths(again), err)
	}
}

// The config takes the binary's keys and keeps the home's features and env, comments and all.
func TestRefreshKeepsTheHomesFeaturesAndEnv(t *testing.T) {
	dir := anInitializedHome(t)
	config := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	data, err := os.ReadFile(config)
	// Stop when the config fails to read
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	features := "{\n        // The tool\n        \"registry.example.com/features/tool:1\": {}\n    }"
	edited := strings.Replace(string(data), `"features": {}`, `"features": `+features, 1)
	edited = strings.Replace(edited, `"name": "o3s"`, `"name": "edited"`, 1)
	edited = strings.Replace(edited, "\n}", ",\n    // The env\n    \"containerEnv\": {\"EXAMPLE_MODE\": \"strict\"}\n}", 1)
	// Stop when the edit fails
	if err := os.WriteFile(config, []byte(edited), 0o600); err != nil { //nolint:gosec // the test writes inside its own temp home
		t.Fatalf("Setup: %v", err)
	}

	got, err := home.Refresh(dir)
	// Stop when the refresh fails
	if err != nil {
		t.Fatalf("Refresh() = %v", err)
	}
	// Stop when the files fail to write
	if err := home.Write(dir, got); err != nil {
		t.Fatalf("Write() = %v", err)
	}

	written, err := os.ReadFile(config)
	// Stop when the config fails to read
	if err != nil {
		t.Fatalf("ReadFile() = %v", err)
	}
	text := string(written)
	// Require the binary's name back, and the home's features and env as they were
	for _, want := range []string{`"name": "o3s"`, `"features": ` + features, "// The env\n    \"containerEnv\": {\"EXAMPLE_MODE\": \"strict\"}"} {
		// Report a part the config lacks
		if !strings.Contains(text, want) {
			t.Errorf("config lacks %q:\n%s", want, text)
		}
	}
}
