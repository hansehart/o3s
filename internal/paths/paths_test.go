package paths_test

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/paths"
)

// Each base folder follows its XDG variable when absolute, and the spec's default otherwise.
func TestBaseFoldersFollowXDG(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	tests := []struct {
		name, value string
		want        []string
	}{
		{"defaults", "", []string{home + "/.config/o3s", home + "/.local/state/o3s", home + "/.local/share/o3s"}},
		{"absolute variables", "/xdg", []string{"/xdg/config/o3s", "/xdg/state/o3s", "/xdg/data/o3s"}},
		{"relative variables, which the spec says to pass over", "xdg", []string{home + "/.config/o3s", home + "/.local/state/o3s", home + "/.local/share/o3s"}},
	}
	// Check every setting of the variables
	for _, tt := range tests {
		// Set each variable where the case names one
		for _, kind := range []string{"CONFIG", "STATE", "DATA"} {
			value := ""
			// Point the variable below the case's base
			if tt.value != "" {
				value = filepath.Join(tt.value, map[string]string{"CONFIG": "config", "STATE": "state", "DATA": "data"}[kind])
			}
			t.Setenv("XDG_"+kind+"_HOME", value)
		}

		got := []string{paths.Config(), paths.State(), paths.Data()}

		// Report folders that differ
		if diff := cmp.Diff(tt.want, got); diff != "" {
			t.Errorf("%s (-want +got):\n%s", tt.name, diff)
		}
	}
}

// The folders o3s keeps per purpose sit below the base folders they belong to.
func TestPurposeFoldersSitBelowTheirBase(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/state")
	t.Setenv("XDG_DATA_HOME", "/data")

	got := []string{paths.Proxy(), paths.Devcontainers()}

	// Report folders that differ
	if diff := cmp.Diff([]string{"/state/o3s/proxy", "/data/o3s/devcontainers"}, got); diff != "" {
		t.Errorf("(-want +got):\n%s", diff)
	}
}

// Each home gets a folder of its own, named the same for the same path on every run.
func TestHomeNamesAFolderPerHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/state")

	a, again, b := paths.Home("/home/ubuntu/o3s"), paths.Home("/home/ubuntu/o3s"), paths.Home("/home/ubuntu/other")

	// Require one folder per home, stable across runs
	if a != again || a == b || filepath.Dir(a) != "/state/o3s" || len(filepath.Base(a)) != 12 {
		t.Errorf("Home() = %q, %q, %q, want one stable 12-character folder per home below /state/o3s", a, again, b)
	}
}
