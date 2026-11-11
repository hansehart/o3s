package update_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/devcontainer"
	"github.com/hansehart/o3s/internal/home"
	"github.com/hansehart/o3s/internal/registry"
	"github.com/hansehart/o3s/internal/registry/registrytest"
	"github.com/hansehart/o3s/internal/update"
)

// toolV2 is the tool's metadata as its store now ships it, needing a new host and new env.
const toolV2 = `{
  "id": "tool",
  "version": "2.0.0",
  "options": {"stateDir": {"type": "string", "default": ""}},
  "customizations": {"o3s": {
    "hosts": [{"name": "example.com", "ports": [443], "secret": "EXAMPLE_TOKEN"}],
    "env": [
      {"name": "EXAMPLE_MODE", "value": "strict"},
      {"name": "EXAMPLE_HOME", "value": "${stateDir}"}
    ]
  }}
}`

// aHomeWithTheTool hands a test a home holding the tool at a tag, as an older store set it up,
// beside a registry whose store now ships toolV2, and the tool's package.
func aHomeWithTheTool(t *testing.T, tag string, userHosts ...string) (string, string) {
	t.Helper()
	r, err := registrytest.New()
	// Stop when the registry fails to come up
	if err != nil {
		t.Fatalf("Setup: registrytest.New() = %v", err)
	}
	t.Cleanup(r.Close)
	// Stop when the store fails to publish
	if err := r.PushFeatures("features", toolV2); err != nil {
		t.Fatalf("Setup: PushFeatures() = %v", err)
	}
	pkg := r.Host + "/features/tool"
	dir := t.TempDir()
	// Stop when the home stays unwritten
	if _, err := home.Init(dir); err != nil {
		t.Fatalf("Setup: Init() = %v", err)
	}
	c, err := config.Load(dir)
	// Stop when the settings fail to load
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Stop when the old host stays out of the allowlist
	if err := c.Need(pkg, config.Change{Name: "example.org"}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Allow every host the user needs
	for _, h := range userHosts {
		// Stop when the user's host stays out
		if _, err := c.Allow(config.Change{Name: h}); err != nil {
			t.Fatalf("Setup: %v", err)
		}
	}
	// Stop when the settings fail to save
	if err := c.Save(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	dc, err := devcontainer.Load(dir)
	// Stop when the config fails to load
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	key := pkg
	// Name the tag where the home carries one
	if tag != "" {
		key += ":" + tag
	}
	dc.SetFeature(pkg, key, nil)
	dc.SetEnv("EXAMPLE_MODE", "lax")
	// Stop when the config fails to save
	if err := dc.Save(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	return dir, pkg
}

// A feature on a major tag moves to the store's major, with the hosts and env it now needs.
func TestCheckFollowsTheStoresMajorVersion(t *testing.T) {
	dir, pkg := aHomeWithTheTool(t, "1")

	got, err := update.Check(t.Context(), dir)

	// Stop when the check fails
	if err != nil {
		t.Fatalf("Check() = %v", err)
	}
	want := []update.FeatureChange{{
		Package:      pkg,
		OldTag:       "1",
		NewTag:       "2",
		HostsAdded:   []registry.Host{{Name: "example.com", Ports: []int{443}, Secret: "EXAMPLE_TOKEN"}},
		HostsDropped: []string{"example.org"},
		EnvAdded:     []registry.Env{{Name: "EXAMPLE_HOME", Value: registry.FeaturesRoot + pkg}},
		EnvChanged:   []update.EnvChange{{Name: "EXAMPLE_MODE", Old: "lax", New: "strict"}},
	}}
	// Report a change that differs
	if diff := cmp.Diff(want, got.Features, cmpopts.IgnoreUnexported(update.FeatureChange{})); diff != "" {
		t.Errorf("Check() features (-want +got):\n%s", diff)
	}
}

// A host the user needs as well stays, so the feature's update drops none.
func TestCheckKeepsAHostTheUserNeeds(t *testing.T) {
	dir, _ := aHomeWithTheTool(t, "1", "example.org")

	got, err := update.Check(t.Context(), dir)

	// Require the feature to drop no host
	if err != nil || len(got.Features) != 1 || len(got.Features[0].HostsDropped) != 0 {
		t.Errorf("Check() = %+v, %v, want one feature that drops no host", got.Features, err)
	}
}

// A tag that pins more than a major version, or follows the newest release, stays as it is.
func TestCheckKeepsAPinnedTag(t *testing.T) {
	// Check every tag beyond a major version
	for _, tag := range []string{"1.2", "latest", ""} {
		dir, _ := aHomeWithTheTool(t, tag)

		got, err := update.Check(t.Context(), dir)

		// Require the tag to stay
		if err != nil || len(got.Features) != 1 || got.Features[0].NewTag != tag {
			t.Errorf("Check() at %q = %+v, %v, want the tag kept", tag, got.Features, err)
		}
	}
}

// A store out of reach leaves its features as they are and is named.
func TestCheckNamesAStoreOutOfReach(t *testing.T) {
	dir := t.TempDir()
	// Stop when the home stays unwritten
	if _, err := home.Init(dir); err != nil {
		t.Fatalf("Setup: Init() = %v", err)
	}
	dc, err := devcontainer.Load(dir)
	// Stop when the config fails to load
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	dc.SetFeature("127.0.0.1:1/features/tool", "127.0.0.1:1/features/tool:1", nil)
	// Stop when the config fails to save
	if err := dc.Save(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	got, err := update.Check(t.Context(), dir)

	// Require nothing to change and the store to be named
	if err != nil || !got.Empty() || len(got.Warnings) != 1 {
		t.Errorf("Check() = %+v, %v, want no change and one warning", got, err)
	}
}

// Applying an update brings the home to what the binary and the stores ship, so a second check finds nothing.
func TestApplyLeavesNothingToUpdate(t *testing.T) {
	dir, pkg := aHomeWithTheTool(t, "1")
	plan, err := update.Check(t.Context(), dir)
	// Stop when the check fails
	if err != nil {
		t.Fatalf("Check() = %v", err)
	}

	hosts, err := plan.Apply(dir)

	// Stop when the update fails
	if err != nil || !hosts {
		t.Fatalf("Apply() = %v, %v, want the hosts changed", hosts, err)
	}
	again, err := update.Check(t.Context(), dir)
	// Require nothing left to update
	if err != nil || !again.Empty() {
		t.Errorf("Check() after Apply = %+v, %v, want nothing", again, err)
	}
	dc, err := devcontainer.Load(dir)
	// Stop when the config fails to load
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	// Require the feature on its new tag
	if diff := cmp.Diff([]string{pkg + ":2"}, dc.Features()); diff != "" {
		t.Errorf("features (-want +got):\n%s", diff)
	}
}
