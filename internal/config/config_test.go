package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/config"
)

// anUnorderedConfig hands a test a config whose hosts and stores sit out of order.
func anUnorderedConfig() config.Config {
	return config.Config{
		Stores: []config.Store{{ID: "registry.example.org/features"}, {ID: "registry.example.com/features"}},
		Hosts: []config.Host{
			{Name: "198.51.100.0/24", Ports: []int{443}, By: []string{"user"}},
			{Name: "api.example.com", Ports: []int{443}, Secret: "EXAMPLE_TOKEN", Inject: []string{"header", "body"}, By: []string{"user"}},
			{Name: "example.com", Ports: []int{22, 443}, By: []string{"user"}},
		},
	}
}

// Saving writes one stable shape, so a diff shows each change alone.
func TestSaveWritesOneStableShape(t *testing.T) {
	home := t.TempDir()
	c := anUnorderedConfig()

	err := c.Save(home)

	// Stop when the file stays unwritten
	if err != nil {
		t.Fatalf("Save(%q) = %v", home, err)
	}
	want, _ := os.ReadFile(filepath.Join("testdata", "ordered.json"))
	got, _ := os.ReadFile(filepath.Join(home, config.File))
	// Require the exact bytes, since the shape is what keeps a diff to one change
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("Save(%q) wrote (-want +got):\n%s", home, diff)
	}
}

// Loading hands back what saving wrote.
func TestLoadReadsWhatSaveWrote(t *testing.T) {
	home := t.TempDir()
	saved := anUnorderedConfig()
	// Stop when the file stays unwritten
	if err := saved.Save(home); err != nil {
		t.Fatalf("Setup: Save(%q) = %v", home, err)
	}

	got, err := config.Load(home)

	// Stop when the file stays unread
	if err != nil {
		t.Fatalf("Load(%q) = %v", home, err)
	}
	// Require the saved config, in its saved order
	if diff := cmp.Diff(saved, got); diff != "" {
		t.Errorf("Load(%q) (-want +got):\n%s", home, diff)
	}
}

// Loading a folder without an o3s home is turned away.
func TestLoadTurnsAwayAFolderWithoutAHome(t *testing.T) {
	home := t.TempDir()

	_, err := config.Load(home)

	// Require the refusal to name the missing home
	if !errors.Is(err, config.ErrNoHome) {
		t.Errorf("Load(%q) = %v, want ErrNoHome", home, err)
	}
}

// Loading a file that holds no valid JSON is turned away.
func TestLoadTurnsAwayAMalformedFile(t *testing.T) {
	home := t.TempDir()
	// Stop when the broken file stays unwritten
	if err := os.WriteFile(filepath.Join(home, config.File), []byte("{"), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	_, err := config.Load(home)

	// Require the broken file to stop the load
	if err == nil {
		t.Errorf("Load(%q) = nil, want an error for the broken file", home)
	}
}
