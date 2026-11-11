package devcontainer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/devcontainer"
)

// aConfig is a config as the user keeps it, with comments, four-space indent and no features.
const aConfig = `// The cage
{
    // Name for the dev container
    "name": "o3s",
    // Additional software layered onto the o3s image
    "features": {},
    // Non-root user VS Code connects as
    "remoteUser": "ubuntu"
}
`

// aHomeWith hands a test an o3s home whose config holds the text given.
func aHomeWith(t *testing.T, text string) string {
	t.Helper()
	home := t.TempDir()
	// Stop when the stack folder fails to appear
	if err := os.MkdirAll(filepath.Join(home, ".devcontainer"), 0o750); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Stop when the config fails to write
	if err := os.WriteFile(filepath.Join(home, devcontainer.File), []byte(text), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	return home
}

// edited loads a home's config, edits it, saves it, and hands back the text it then holds.
func edited(t *testing.T, text string, edit func(*devcontainer.Config)) string {
	t.Helper()
	home := aHomeWith(t, text)
	c, err := devcontainer.Load(home)
	// Stop when the config fails to load
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	edit(c)
	// Stop when the config fails to save
	if err := c.Save(home); err != nil {
		t.Fatalf("Save() = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(home, devcontainer.File))
	// Stop when the saved config fails to read
	if err != nil {
		t.Fatalf("ReadFile() = %v", err)
	}
	return string(data)
}

// A feature joins the features in the config's own layout, its comments kept.
func TestSetFeatureKeepsTheLayout(t *testing.T) {
	got := edited(t, aConfig, func(c *devcontainer.Config) {
		c.SetFeature("registry.example.com/features/tool", "registry.example.com/features/tool:1", map[string]string{"stateDir": "/home/ubuntu/state/registry.example.com/features/tool"})
		c.SetFeature("registry.example.com/features/other", "registry.example.com/features/other:2", nil)
	})

	want := `// The cage
{
    // Name for the dev container
    "name": "o3s",
    // Additional software layered onto the o3s image
    "features": {
        "registry.example.com/features/tool:1": {
            "stateDir": "/home/ubuntu/state/registry.example.com/features/tool"
        },
        "registry.example.com/features/other:2": {}
    },
    // Non-root user VS Code connects as
    "remoteUser": "ubuntu"
}
`
	// Report a layout that differs
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("config (-want +got):\n%s", diff)
	}
}

// A feature set again under another tag takes the place of the one it updates.
func TestSetFeatureReplacesTheSamePackageInPlace(t *testing.T) {
	text := `{
    "features": {
        // The tool
        "registry.example.com/features/tool:1": {},
        "registry.example.com/features/other:2": {}
    }
}
`
	got := edited(t, text, func(c *devcontainer.Config) {
		c.SetFeature("registry.example.com/features/tool", "registry.example.com/features/tool:2", nil)
	})

	want := `{
    "features": {
        // The tool
        "registry.example.com/features/tool:2": {},
        "registry.example.com/features/other:2": {}
    }
}
`
	// Report a config that differs
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("config (-want +got):\n%s", diff)
	}
}

// Removing a feature drops it whatever tag it carries, and reports whether it was there.
func TestRemoveFeatureDropsThePackage(t *testing.T) {
	text := `{
    "features": {
        "registry.example.com/features/tool:1": {},
        "registry.example.com/features/tools:2": {}
    }
}
`
	var removed, again bool
	got := edited(t, text, func(c *devcontainer.Config) {
		removed = c.RemoveFeature("registry.example.com/features/tool")
		again = c.RemoveFeature("registry.example.com/features/tool")
	})

	want := `{
    "features": {
        "registry.example.com/features/tools:2": {}
    }
}
`
	// Report a config that differs
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("config (-want +got):\n%s", diff)
	}
	// Require the first removal alone to report the feature
	if !removed || again {
		t.Errorf("RemoveFeature twice = %v, %v, want true, false", removed, again)
	}
}

// Env lands in the container's env, which the config gains where it lacks it.
func TestSetEnvAddsTheContainersEnv(t *testing.T) {
	got := edited(t, aConfig, func(c *devcontainer.Config) {
		c.SetEnv("EXAMPLE_MODE", "lax")
		c.SetEnv("EXAMPLE_MODE", "strict")
		c.SetEnv("EXAMPLE_HOME", "/home/ubuntu/state")
	})

	want := `// The cage
{
    // Name for the dev container
    "name": "o3s",
    // Additional software layered onto the o3s image
    "features": {},
    // Non-root user VS Code connects as
    "remoteUser": "ubuntu",
    "containerEnv": {
        "EXAMPLE_MODE": "strict",
        "EXAMPLE_HOME": "/home/ubuntu/state"
    }
}
`
	// Report a config that differs
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("config (-want +got):\n%s", diff)
	}
}

// Removing env drops the variable and keeps the rest.
func TestRemoveEnvDropsTheVariable(t *testing.T) {
	text := `{
    "containerEnv": {
        "EXAMPLE_MODE": "strict",
        "EXAMPLE_HOME": "/home/ubuntu/state"
    }
}
`
	got := edited(t, text, func(c *devcontainer.Config) { c.RemoveEnv("EXAMPLE_MODE") })

	want := `{
    "containerEnv": {
        "EXAMPLE_HOME": "/home/ubuntu/state"
    }
}
`
	// Report a config that differs
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("config (-want +got):\n%s", diff)
	}
}

// The features come back as the config names them, in its order.
func TestFeaturesListsTheConfigsFeatures(t *testing.T) {
	home := aHomeWith(t, `{"features": {"registry.example.com/features/tool:1": {}, "registry.example.com/features/other:2": {"mode": "strict"}}}`)
	c, err := devcontainer.Load(home)
	// Stop when the config fails to load
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	got := c.Features()

	// Report features that differ
	if diff := cmp.Diff([]string{"registry.example.com/features/tool:1", "registry.example.com/features/other:2"}, got); diff != "" {
		t.Errorf("Features() (-want +got):\n%s", diff)
	}
}

// A config that holds no object is turned away.
func TestLoadTurnsAwayABrokenConfig(t *testing.T) {
	home := aHomeWith(t, `{"features": `)

	_, err := devcontainer.Load(home)

	// Require an error for a broken config
	if err == nil {
		t.Errorf("Load() = nil, want an error")
	}
}
