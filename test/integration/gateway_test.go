//go:build integration

// Package integration_test covers one real component of o3s at a time, built from this branch.
package integration_test

import (
	"strings"
	"testing"

	"github.com/hansehart/o3s/test/stacktest"
)

// An empty policy raises a gateway that reports healthy.
func TestTheGatewayComesUpHealthyOnAnEmptyPolicy(t *testing.T) {
	stacktest.AnEmptyPolicy(t)

	err := stacktest.Up(t, "gateway")

	// Report a gateway that fails to report healthy
	if err != nil {
		t.Errorf("Up(gateway) on an empty policy = %v, want the gateway healthy", err)
	}
}

// On an empty policy the gateway forwards replies alone and rejects every new connection.
func TestTheGatewayRejectsEveryNewConnectionOnAnEmptyPolicy(t *testing.T) {
	stacktest.AnEmptyPolicy(t)
	// Stop when the gateway fails to come up
	if err := stacktest.Up(t, "gateway"); err != nil {
		t.Fatalf("Setup: Up(gateway): %v", err)
	}

	out, err := stacktest.Exec(t, "gateway", "iptables", "-S", "FORWARD")

	// Stop when reading the rules fails
	if err != nil {
		t.Fatalf("Exec(gateway, iptables -S FORWARD) = %v", err)
	}
	rules := strings.Split(strings.TrimSpace(out), "\n")
	// Check every rule in the chain
	for _, rule := range rules {
		// Flag an accepting rule that admits new connections
		if strings.HasSuffix(rule, "-j ACCEPT") && !strings.Contains(rule, "--ctstate") {
			t.Errorf("FORWARD accepts %q, want replies alone accepted", rule)
		}
	}
	// Require the chain to close with a reject
	if got, want := rules[len(rules)-1], "-A FORWARD -j REJECT --reject-with icmp-admin-prohibited"; got != want {
		t.Errorf("last FORWARD rule = %q, want %q", got, want)
	}
}

// The gateway opens every port the home's policy names for a host.
func TestTheGatewayOpensThePortsTheHomesPolicyNames(t *testing.T) {
	stacktest.AnEmptyPolicy(t)
	stacktest.AHomePolicy(t, `{"hosts": [{"name": "github.com", "ports": [22, 443], "by": ["user"]}]}`)
	// Stop when the gateway fails to come up
	if err := stacktest.Up(t, "gateway"); err != nil {
		t.Fatalf("Setup: Up(gateway): %v", err)
	}

	out, err := stacktest.Exec(t, "gateway", "iptables", "-S", "FORWARD")

	// Stop when reading the rules fails
	if err != nil {
		t.Fatalf("Exec(gateway, iptables -S FORWARD) = %v", err)
	}
	// Require an accepting rule for every port the policy names
	for _, port := range []string{"22", "443"} {
		// Flag a named port the chain leaves closed
		if !acceptsPort(out, port) {
			t.Errorf("FORWARD accepts nothing on port %s from its set, rules:\n%s", port, out)
		}
	}
}

// acceptsPort reports whether a chain holds a rule that accepts a port for the addresses in its set.
func acceptsPort(rules, port string) bool {
	// Look for one rule that names the port's set, the port and the accept
	for rule := range strings.SplitSeq(rules, "\n") {
		// Report the first rule that opens the port
		if strings.Contains(rule, "--match-set allowed-p"+port+" dst") &&
			strings.Contains(rule, "--dport "+port+" ") &&
			strings.HasSuffix(rule, "-j ACCEPT") {
			return true
		}
	}
	return false
}

// The gateway follows the home's policy wherever the old policy still sits beside it.
func TestTheGatewayPrefersTheHomesPolicyOverTheOldOne(t *testing.T) {
	stacktest.AnOldPolicy(t, "[\"github.com\"]\nports = [8443]\n")
	stacktest.AHomePolicy(t, `{"hosts": []}`)
	// Stop when the gateway fails to come up
	if err := stacktest.Up(t, "gateway"); err != nil {
		t.Fatalf("Setup: Up(gateway): %v", err)
	}

	out, err := stacktest.Exec(t, "gateway", "iptables", "-S", "FORWARD")

	// Stop when reading the rules fails
	if err != nil {
		t.Fatalf("Exec(gateway, iptables -S FORWARD) = %v", err)
	}
	// Require the old policy's port to stay closed
	if strings.Contains(out, "--dport 8443") {
		t.Errorf("FORWARD opens 8443 from the old policy, rules:\n%s", out)
	}
}
