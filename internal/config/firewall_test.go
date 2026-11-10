package config_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/config"
)

// aConfigThatAllows hands a test a config holding the given hosts.
func aConfigThatAllows(hosts ...config.Host) config.Config {
	return config.Config{Hosts: hosts}
}

// A new host takes port 443, the header for its secret, and the user as the one who needs it.
func TestAllowGivesANewHostItsDefaults(t *testing.T) {
	var c config.Config

	diffs, err := c.Allow(config.Change{Name: "api.openai.com", Secret: "OPENAI_API_KEY"})

	// Stop when the host is turned away
	if err != nil {
		t.Fatalf("Allow(api.openai.com) = %v, want it allowed", err)
	}
	want := []config.Host{{Name: "api.openai.com", Ports: []int{443}, Secret: "OPENAI_API_KEY", Inject: []string{"header"}, By: []string{"user"}}}
	// Require the defaults on the new host
	if diff := cmp.Diff(want, c.Hosts); diff != "" {
		t.Errorf("Allow(api.openai.com) hosts (-want +got):\n%s", diff)
	}
	// Require a new host to report no changed fields
	if len(diffs) != 0 {
		t.Errorf("Allow(api.openai.com) diffs = %v, want none for a new host", diffs)
	}
}

// Ports arrive deduplicated and in order, whatever order they were given in.
func TestAllowOrdersAndDeduplicatesPorts(t *testing.T) {
	var c config.Config

	_, err := c.Allow(config.Change{Name: "github.com", Ports: []int{443, 22, 443}})

	// Stop when the host is turned away
	if err != nil {
		t.Fatalf("Allow(github.com) = %v", err)
	}
	// Require each port once, in order
	if diff := cmp.Diff([]int{22, 443}, c.Hosts[0].Ports); diff != "" {
		t.Errorf("Allow(github.com) ports (-want +got):\n%s", diff)
	}
}

// Updating a host replaces the fields given as a whole, keeps the rest, and reports each change.
func TestAllowUpdatesTheFieldsGivenAndKeepsTheRest(t *testing.T) {
	c := aConfigThatAllows(config.Host{Name: "api.openai.com", Ports: []int{443}, Secret: "OPENAI_API_KEY", Inject: []string{"header"}, By: []string{"user"}})

	diffs, err := c.Allow(config.Change{Name: "api.openai.com", Ports: []int{443, 8443}})

	// Stop when the update is turned away
	if err != nil {
		t.Fatalf("Allow(api.openai.com) = %v", err)
	}
	want := []config.Host{{Name: "api.openai.com", Ports: []int{443, 8443}, Secret: "OPENAI_API_KEY", Inject: []string{"header"}, By: []string{"user"}}}
	// Require the new ports beside the kept secret and inject sites
	if diff := cmp.Diff(want, c.Hosts); diff != "" {
		t.Errorf("Allow(api.openai.com) hosts (-want +got):\n%s", diff)
	}
	// Require the ports, and the ports alone, in the report
	if diff := cmp.Diff([]config.Diff{{Field: "ports", Old: "443", New: "443,8443"}}, diffs); diff != "" {
		t.Errorf("Allow(api.openai.com) diffs (-want +got):\n%s", diff)
	}
}

// Updating a host a feature brought makes it the user's as well, so removing the feature keeps it.
func TestAllowAddsTheUserToAFeaturesHost(t *testing.T) {
	feature := "ghcr.io/hansehart/devcontainer-features/claude-code"
	c := aConfigThatAllows(config.Host{Name: "api.anthropic.com", Ports: []int{443}, By: []string{feature}})

	diffs, err := c.Allow(config.Change{Name: "api.anthropic.com"})

	// Stop when the update is turned away
	if err != nil {
		t.Fatalf("Allow(api.anthropic.com) = %v", err)
	}
	// Require the user beside the feature
	if diff := cmp.Diff([]string{feature, "user"}, c.Hosts[0].By); diff != "" {
		t.Errorf("Allow(api.anthropic.com) by (-want +got):\n%s", diff)
	}
	// Require the report to show who needs the host now
	if diff := cmp.Diff([]config.Diff{{Field: "by", Old: feature, New: feature + ",user"}}, diffs); diff != "" {
		t.Errorf("Allow(api.anthropic.com) diffs (-want +got):\n%s", diff)
	}
}

// Hosts follow their apex domain, the apex before its subdomains, with IPs and ranges last.
func TestAllowOrdersHostsByApex(t *testing.T) {
	var c config.Config
	names := []string{"10.20.0.0/16", "api.openai.com", "api.github.com", "*.docker.io", "github.com", "api.anthropic.com", "10.0.0.5"}

	// Allow every host, in an order the config has to put right
	for _, name := range names {
		// Stop when a host is turned away
		if _, err := c.Allow(config.Change{Name: name}); err != nil {
			t.Fatalf("Allow(%q) = %v", name, err)
		}
	}

	got := make([]string, 0, len(c.Hosts))
	// Read back the order the config holds
	for _, h := range c.Hosts {
		got = append(got, h.Name)
	}
	want := []string{"api.anthropic.com", "*.docker.io", "github.com", "api.github.com", "api.openai.com", "10.0.0.5", "10.20.0.0/16"}
	// Require the apex order
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("host order (-want +got):\n%s", diff)
	}
}

// Removing a host the user alone needs takes it off the allowlist.
func TestRemoveDropsAHostTheUserAloneNeeds(t *testing.T) {
	c := aConfigThatAllows(config.Host{Name: "github.com", Ports: []int{22, 443}, By: []string{"user"}})

	kept, err := c.Remove("github.com")

	// Stop when the removal is turned away
	if err != nil {
		t.Fatalf("Remove(github.com) = %v", err)
	}
	// Require the host gone
	if len(c.Hosts) != 0 || kept != nil {
		t.Errorf("Remove(github.com) = kept %v, hosts %v, want the host gone", kept, c.Hosts)
	}
}

// Removing a host a feature still needs keeps it, and names the feature.
func TestRemoveKeepsAHostAFeatureStillNeeds(t *testing.T) {
	feature := "ghcr.io/hansehart/devcontainer-features/codex"
	c := aConfigThatAllows(config.Host{Name: "api.openai.com", Ports: []int{443}, By: []string{feature, "user"}})

	kept, err := c.Remove("api.openai.com")

	// Stop when the removal is turned away
	if err != nil {
		t.Fatalf("Remove(api.openai.com) = %v", err)
	}
	// Require the feature named as the one still in need
	if diff := cmp.Diff([]string{feature}, kept); diff != "" {
		t.Errorf("Remove(api.openai.com) kept (-want +got):\n%s", diff)
	}
	// Require the feature alone to hold the host
	if diff := cmp.Diff([]string{feature}, c.Hosts[0].By); diff != "" {
		t.Errorf("Remove(api.openai.com) by (-want +got):\n%s", diff)
	}
}

// Removing a host only features brought is turned away, since the features own it.
func TestRemoveTurnsAwayAHostOnlyFeaturesBrought(t *testing.T) {
	c := aConfigThatAllows(config.Host{Name: "api.anthropic.com", Ports: []int{443}, By: []string{"ghcr.io/hansehart/devcontainer-features/claude-code"}})

	_, err := c.Remove("api.anthropic.com")

	// Require the refusal to point at the features
	if !errors.Is(err, config.ErrOwnedByFeatures) {
		t.Errorf("Remove(api.anthropic.com) = %v, want ErrOwnedByFeatures", err)
	}
}

// Removing a host the allowlist lacks is turned away.
func TestRemoveTurnsAwayAnUnknownHost(t *testing.T) {
	var c config.Config

	_, err := c.Remove("github.com")

	// Require the refusal to name the unknown host
	if !errors.Is(err, config.ErrUnknownHost) {
		t.Errorf("Remove(github.com) = %v, want ErrUnknownHost", err)
	}
}
