package stacktest

import (
	"os"
	"path/filepath"
	"testing"
)

// The profile the cage runs under, where the kernel lays out what it holds, and where the host
// keeps the profile it loads at boot.
const (
	profileName      = "o3s-cage"
	kernelPolicy     = "/sys/kernel/security/apparmor/policy/profiles"
	installedProfile = "/etc/apparmor.d/" + profileName
)

// AHostThatLostTheProfile leaves the host holding the cage's installed profile alone, as a restart
// that empties the kernel does. It raises the stack first, so the profile the kernel then loses is
// one the stack loaded.
func AHostThatLostTheProfile(tb testing.TB) {
	tb.Helper()
	raised, err := DevcontainerUp(tb)
	// Stop when the stack fails to come up
	if err != nil {
		tb.Fatalf("Setup: %v", err)
	}
	// Require a loaded profile first, so its absence below carries weight
	if !profileIsLoaded(tb) {
		tb.Fatalf("Setup: the kernel holds no %s profile after the stack came up", profileName)
	}
	// Stop when the cage survives its removal
	if _, err := run(tb.Context(), tb, "docker", "rm", "-f", raised.ContainerID); err != nil {
		tb.Fatalf("Setup: %v", err)
	}
	// Stop when the kernel keeps the profile
	if _, err := run(tb.Context(), tb, "sudo", "apparmor_parser", "-R", filepath.Join(Repo(tb), ".devcontainer/apparmor.conf")); err != nil {
		tb.Fatalf("Setup: %v", err)
	}
	// Hold the arranged state to what it claims, so the test reads a kernel that lost the profile
	if profileIsLoaded(tb) {
		tb.Fatalf("Setup: the kernel still holds the %s profile", profileName)
	}
	// Require the installed profile the next start loads from
	if _, err := os.Stat(installedProfile); err != nil {
		tb.Fatalf("Setup: find the installed profile: %v", err)
	}
}

// profileIsLoaded reads the policy the kernel lays out, where every load names the profile afresh.
func profileIsLoaded(tb testing.TB) bool {
	tb.Helper()
	loaded, err := filepath.Glob(filepath.Join(kernelPolicy, profileName+".*"))
	// Stop when the pattern for the profile is malformed
	if err != nil {
		tb.Fatalf("Setup: read the kernel's policy: %v", err)
	}
	return len(loaded) > 0
}
