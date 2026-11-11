package config_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/config"
)

// tool is the package of a feature the fixtures install.
const tool = "registry.example.com/features/tool"

// A host a feature needs joins the allowlist with the feature as its reason and its defaults.
func TestNeedAddsAHostForAFeature(t *testing.T) {
	var c config.Config

	err := c.Need(tool, config.Change{Name: "example.com", Secret: "EXAMPLE_TOKEN"})

	// Stop when the host is turned away
	if err != nil {
		t.Fatalf("Need(example.com) = %v", err)
	}
	want := []config.Host{{Name: "example.com", Ports: []int{443}, Secret: "EXAMPLE_TOKEN", Inject: []string{"header"}, By: []string{tool}}}
	// Require the host with its defaults and the feature as its reason
	if diff := cmp.Diff(want, c.Hosts); diff != "" {
		t.Errorf("hosts (-want +got):\n%s", diff)
	}
}

// A host the user already allows keeps its ports, gains the feature's, and lists both reasons.
func TestNeedJoinsAHostTheUserAllows(t *testing.T) {
	c := aConfigThatAllows(config.Host{Name: "example.com", Ports: []int{22}, By: []string{"user"}})

	err := c.Need(tool, config.Change{Name: "example.com", Ports: []int{443}, Secret: "EXAMPLE_TOKEN", Inject: []string{"body"}})

	// Stop when the host is turned away
	if err != nil {
		t.Fatalf("Need(example.com) = %v", err)
	}
	want := []config.Host{{Name: "example.com", Ports: []int{22, 443}, Secret: "EXAMPLE_TOKEN", Inject: []string{"body"}, By: []string{"user", tool}}}
	// Require the joined host
	if diff := cmp.Diff(want, c.Hosts); diff != "" {
		t.Errorf("hosts (-want +got):\n%s", diff)
	}
}

// Needing a host twice keeps one reason for it.
func TestNeedListsAFeatureOnce(t *testing.T) {
	var c config.Config

	// Need the same host twice
	for range 2 {
		// Stop when the host is turned away
		if err := c.Need(tool, config.Change{Name: "example.com"}); err != nil {
			t.Fatalf("Need(example.com) = %v", err)
		}
	}

	// Require the feature listed once
	if diff := cmp.Diff([]string{tool}, c.Hosts[0].By); diff != "" {
		t.Errorf("by (-want +got):\n%s", diff)
	}
}

// A host that already carries another secret is turned away and stays as it was.
func TestNeedTurnsAwayASecondSecret(t *testing.T) {
	host := config.Host{Name: "example.com", Ports: []int{443}, Secret: "EXAMPLE_TOKEN", Inject: []string{"header"}, By: []string{"user"}}
	c := aConfigThatAllows(host)

	err := c.Need(tool, config.Change{Name: "example.com", Secret: "OTHER_TOKEN"})

	// Require the error that marks a broken rule
	if !errors.Is(err, config.ErrInvalid) {
		t.Errorf("Need(example.com) = %v, want ErrInvalid", err)
	}
	// Require the host as it was
	if diff := cmp.Diff([]config.Host{host}, c.Hosts); diff != "" {
		t.Errorf("hosts (-want +got):\n%s", diff)
	}
}

// A host a feature names in a form the gateway lacks is turned away.
func TestNeedTurnsAwayAMalformedHost(t *testing.T) {
	var c config.Config

	err := c.Need(tool, config.Change{Name: "Example.com"})

	// Require the error that marks a broken rule
	if !errors.Is(err, config.ErrInvalid) {
		t.Errorf("Need(Example.com) = %v, want ErrInvalid", err)
	}
}

// Releasing a feature drops the hosts it alone needs and keeps the rest for their other reasons.
func TestReleaseDropsWhatTheFeatureAloneNeeds(t *testing.T) {
	c := aConfigThatAllows(
		config.Host{Name: "example.com", Ports: []int{443}, By: []string{tool}},
		config.Host{Name: "example.org", Ports: []int{22, 443}, By: []string{"user", tool}},
		config.Host{Name: "example.net", Ports: []int{443}, By: []string{"user"}},
	)

	dropped := c.Release(tool)

	want := []config.Host{
		{Name: "example.org", Ports: []int{22, 443}, By: []string{"user"}},
		{Name: "example.net", Ports: []int{443}, By: []string{"user"}},
	}
	// Require the hosts the others still need
	if diff := cmp.Diff(want, c.Hosts); diff != "" {
		t.Errorf("hosts (-want +got):\n%s", diff)
	}
	// Require the dropped host to be named
	if diff := cmp.Diff([]string{"example.com"}, dropped); diff != "" {
		t.Errorf("Release() (-want +got):\n%s", diff)
	}
}
