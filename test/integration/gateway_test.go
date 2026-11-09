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

	if err != nil {
		t.Errorf("Up(gateway) on an empty policy = %v, want the gateway healthy", err)
	}
}

// On an empty policy the gateway forwards replies alone and rejects every new connection.
func TestTheGatewayRejectsEveryNewConnectionOnAnEmptyPolicy(t *testing.T) {
	stacktest.AnEmptyPolicy(t)
	if err := stacktest.Up(t, "gateway"); err != nil {
		t.Fatalf("Setup: Up(gateway): %v", err)
	}

	out, err := stacktest.Exec(t, "gateway", "iptables", "-S", "FORWARD")

	if err != nil {
		t.Fatalf("Exec(gateway, iptables -S FORWARD) = %v", err)
	}
	rules := strings.Split(strings.TrimSpace(out), "\n")
	for _, rule := range rules {
		if strings.HasSuffix(rule, "-j ACCEPT") && !strings.Contains(rule, "--ctstate") {
			t.Errorf("FORWARD accepts %q, want replies alone accepted", rule)
		}
	}
	if got, want := rules[len(rules)-1], "-A FORWARD -j REJECT --reject-with icmp-admin-prohibited"; got != want {
		t.Errorf("last FORWARD rule = %q, want %q", got, want)
	}
}
