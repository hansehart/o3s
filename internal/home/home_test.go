package home_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tailscale/hujson"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/home"
)

// anInitializedHome hands a test a folder o3s has just made into a home.
func anInitializedHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Stop when the home stays unwritten
	if _, err := home.Init(dir); err != nil {
		t.Fatalf("Setup: Init(%q) = %v", dir, err)
	}
	return dir
}

// readConfig reads a devcontainer config, comments and all, as plain JSON values.
func readConfig(t *testing.T, data []byte) map[string]any {
	t.Helper()
	plain, err := hujson.Standardize(data)
	// Stop when the config holds no JSON with comments
	if err != nil {
		t.Fatalf("Standardize: %v", err)
	}
	var v map[string]any
	// Stop when the config reads in another shape than an object
	if err := json.Unmarshal(plain, &v); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return v
}

// A new home holds the settings of a new home, both stores and no host.
func TestInitWritesTheSettingsOfANewHome(t *testing.T) {
	dir := t.TempDir()

	_, err := home.Init(dir)

	// Stop when the home stays unwritten
	if err != nil {
		t.Fatalf("Init(%q) = %v", dir, err)
	}
	got, err := config.Load(dir)
	// Stop when the settings stay unread
	if err != nil {
		t.Fatalf("Load(%q) = %v", dir, err)
	}
	want := config.New()
	want.Schema = got.Schema
	// Require the settings of a new home
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Init(%q) settings (-want +got):\n%s", dir, diff)
	}
}

// A new home carries every file of the stack as the binary holds it, its own config aside.
func TestInitCopiesTheStack(t *testing.T) {
	dir := anInitializedHome(t)

	err := fs.WalkDir(home.Files, "stack", func(path string, d fs.DirEntry, err error) error {
		// Hand back a walk that fails
		if err != nil || d.IsDir() || filepath.Base(path) == "devcontainer.json" {
			return err
		}
		want, _ := home.Files.ReadFile(path)
		got, readErr := os.ReadFile(filepath.Join(dir, ".devcontainer", strings.TrimPrefix(path, "stack/")))
		// Flag a file the home lacks or holds otherwise
		if readErr != nil || !bytes.Equal(want, got) {
			t.Errorf("home file %s = %v, differs from the stack the binary holds", path, readErr)
		}
		return nil
	})

	// Stop when the stack stays unwalked
	if err != nil {
		t.Fatalf("WalkDir = %v", err)
	}
}

// A new home's config keeps the stack's settings and suggests no feature.
func TestInitStartsTheConfigWithoutFeatures(t *testing.T) {
	dir := anInitializedHome(t)
	source, _ := home.Files.ReadFile("stack/devcontainer.json")

	written, err := os.ReadFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"))

	// Stop when the config stays unread
	if err != nil {
		t.Fatalf("read the home's config: %v", err)
	}
	want, got := readConfig(t, source), readConfig(t, written)
	want["features"] = map[string]any{}
	// Require every setting of the stack with the features emptied
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("home config (-want +got):\n%s", diff)
	}
}

// A folder that already holds a home is turned away, and keeps every file as it was.
func TestInitTurnsAwayAnExistingHome(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, config.File)
	// Stop when the earlier settings stay unwritten
	if err := os.WriteFile(settings, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	_, err := home.Init(dir)

	// Require the refusal to name the existing home
	if !errors.Is(err, home.ErrExists) {
		t.Errorf("Init(%q) = %v, want ErrExists", dir, err)
	}
	kept, _ := os.ReadFile(settings)
	_, statErr := os.Stat(filepath.Join(dir, ".devcontainer"))
	// Require the earlier settings kept and the stack left out
	if string(kept) != "{}\n" || !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("Init(%q) touched the folder: settings %q, stack %v", dir, kept, statErr)
	}
}

// A release writes its own version as the tag of the o3s images, and any other build keeps latest.
func TestInitPinsTheImagesToTheRelease(t *testing.T) {
	tests := []struct {
		version string
		tag     string
	}{
		{"0.7.0", "0.7.0"},
		{"dev", "latest"},
		{"0.6.6-SNAPSHOT-03605dd", "latest"},
	}
	// Check every kind of build
	for _, tt := range tests {
		home.Version = tt.version
		t.Cleanup(func() { home.Version = "dev" })
		dir := anInitializedHome(t)

		data, err := os.ReadFile(filepath.Join(dir, ".devcontainer", "compose.yaml"))

		// Stop when the compose file fails to read
		if err != nil {
			t.Fatalf("ReadFile() = %v", err)
		}
		// Require both o3s images at the tag
		for _, image := range []string{"ghcr.io/hansehart/o3s-base:", "ghcr.io/hansehart/o3s-gateway:"} {
			// Report an image at another tag
			if !bytes.Contains(data, []byte("image: "+image+tt.tag+"\n")) {
				t.Errorf("version %s: compose lacks %s%s", tt.version, image, tt.tag)
			}
		}
	}
}

// A home holds the stack alone, while the templates the binary seeds from stay in the binary.
func TestInitLeavesTheTemplatesInTheBinary(t *testing.T) {
	dir := anInitializedHome(t)

	_, err := os.Stat(filepath.Join(dir, ".devcontainer", "templates"))

	// Require the home without a copy of the templates
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("home holds .devcontainer/templates (%v), want it absent", err)
	}
}
