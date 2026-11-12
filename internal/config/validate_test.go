package config_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/config"
)

// A change outside the allowlist's rules leaves the config as it was and names the rule it broke.
func TestAllowRejectsAChangeOutsideTheRules(t *testing.T) {
	tests := []struct {
		name   string
		change config.Change
	}{
		{name: "uppercase host", change: config.Change{Name: "Example.com"}},
		{name: "host with a scheme", change: config.Change{Name: "https://example.com"}},
		{name: "wildcard inside a label", change: config.Change{Name: "api.*.example.com"}},
		{name: "IPv6 address", change: config.Change{Name: "::1"}},
		{name: "CIDR beyond 32 bits", change: config.Change{Name: "192.0.2.0/33"}},
		{name: "octet past 255", change: config.Change{Name: "256.0.0.1"}},
		{name: "address short of four octets", change: config.Change{Name: "192.0.2"}},
		{name: "top-level label of digits alone", change: config.Change{Name: "example.123"}},
		{name: "port zero", change: config.Change{Name: "example.com", Ports: []int{0}}},
		{name: "port past the range", change: config.Change{Name: "example.com", Ports: []int{65536}}},
		{name: "lowercase secret", change: config.Change{Name: "api.example.com", Secret: "example_token"}},
		{name: "unknown inject site", change: config.Change{Name: "api.example.com", Secret: "EXAMPLE_TOKEN", Inject: []string{"cookie"}}},
		{name: "inject without a secret", change: config.Change{Name: "api.example.com", Inject: []string{"header"}}},
	}
	// Check every change against the same empty config
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c config.Config

			_, err := c.Allow(tt.change)

			// Require the rejection to name the broken rule
			if !errors.Is(err, config.ErrInvalid) {
				t.Errorf("Allow(%+v) = %v, want ErrInvalid", tt.change, err)
			}
			// Require the config to stay as it was
			if diff := cmp.Diff(config.Config{}, c); diff != "" {
				t.Errorf("Allow(%+v) changed the config (-want +got):\n%s", tt.change, diff)
			}
		})
	}
}

// A change inside the allowlist's rules goes through, whatever form of host it names.
func TestAllowAcceptsEveryFormOfHost(t *testing.T) {
	names := []string{"example.com", "*.example.org", "192.0.2.1", "198.51.100.0/24", "example1.com", "1example.com", "xn--bcher-kva.example", "localhost"}
	// Check every form against its own empty config
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			var c config.Config

			_, err := c.Allow(config.Change{Name: name})

			// Report a form the rules turned away
			if err != nil {
				t.Errorf("Allow(%q) = %v, want it allowed", name, err)
			}
		})
	}
}
