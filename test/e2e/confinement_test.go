//go:build e2e

// Package e2e_test covers the whole o3s stack on a real host, raised the way the editor raises it.
package e2e_test

import (
	"strings"
	"testing"

	"github.com/hansehart/o3s/test/stacktest"
)

// A host whose kernel lost the cage's profile still raises a cage the kernel confines.
func TestTheCageIsConfinedAfterTheKernelLosesTheProfile(t *testing.T) {
	stacktest.AHostThatLostTheProfile(t)

	raised, err := stacktest.DevcontainerUp(t)

	// Stop when the stack fails to come up
	if err != nil {
		t.Fatalf("DevcontainerUp() = %v, want the stack raised", err)
	}
	out, err := stacktest.ContainerExec(t, raised.ContainerID, "cat", "/proc/self/attr/current")
	// Stop when reading the confinement fails
	if err != nil {
		t.Fatalf("ContainerExec(cage, cat /proc/self/attr/current) = %v", err)
	}
	// Require the cage to run under its profile in enforce mode
	if got, want := strings.TrimSpace(out), "o3s-cage (enforce)"; got != want {
		t.Errorf("cage confinement = %q, want %q", got, want)
	}
}
