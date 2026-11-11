package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/menu"
)

// The menu knows every secret the home names or stores, whether it holds a value, and the home's stores.
func TestMenuHomeReadsTheHome(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	secretsDir := t.TempDir()
	t.Setenv("O3S_SECRETS_DIR", secretsDir)
	settings := `{"$schema": "./.devcontainer/o3s.schema.json", "stores": [{"id": "registry.example.com/features"}], "hosts": [
  {"name": "example.com", "ports": [443], "secret": "EXAMPLE_TOKEN", "inject": ["header"], "by": ["user"]}
]}`
	// Stop when the settings stay unwritten
	if err := os.WriteFile(filepath.Join(dir, "o3s.json"), []byte(settings), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Stop when the secrets stay unwritten
	if err := os.WriteFile(filepath.Join(secretsDir, "secrets.env"), []byte("OTHER_TOKEN=value\n"), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	got := menuHome()

	want := menu.Home{
		Dir:     dir,
		Hosts:   got.Hosts,
		Secrets: []menu.Secret{{Name: "EXAMPLE_TOKEN"}, {Name: "OTHER_TOKEN", Set: true}},
		Stores:  []string{"registry.example.com/features"},
	}
	// Report a home read otherwise
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("menuHome() (-want +got):\n%s", diff)
	}
	// Require the host the settings name
	if len(got.Hosts) != 1 || got.Hosts[0].Name != "example.com" {
		t.Errorf("menuHome() hosts = %v, want example.com", got.Hosts)
	}
}

// Outside a home the menu still starts, knowing only where it runs.
func TestMenuHomeStartsOutsideAHome(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("O3S_SECRETS_DIR", t.TempDir())

	got := menuHome()

	// Require a home that names the folder alone
	if diff := cmp.Diff(menu.Home{Dir: dir}, got); diff != "" {
		t.Errorf("menuHome() (-want +got):\n%s", diff)
	}
}
