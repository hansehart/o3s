package stacktest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The stack built from this branch and where the cage's profile lives.
const (
	devProject       = "o3s-dev"
	devGateway       = "o3s-gateway:dev"
	profileName      = "o3s-cage"
	kernelPolicy     = "/sys/kernel/security/apparmor/policy/profiles"
	installedProfile = "/etc/apparmor.d/" + profileName
)

// AHostThatLostTheProfile leaves the cage's profile installed but unloaded, as after a reboot. It
// raises the stack first, so the profile the kernel loses is one the stack loaded.
func AHostThatLostTheProfile(tb testing.TB) {
	tb.Helper()
	raised, err := DevcontainerUp(tb)
	// Stop when the stack fails to come up
	if err != nil {
		tb.Fatalf("Setup: %v", err)
	}
	gateways, err := run(tb.Context(), tb, "docker", "ps", "--filter", "label=com.docker.compose.service=gateway", "--format", "{{.Label \"com.docker.compose.project\"}} {{.Image}}")
	// Stop when the running gateways stay unread
	if err != nil {
		tb.Fatalf("Setup: %v", err)
	}
	// Require the gateway built from this branch, so the test covers the code under review
	if !strings.Contains(gateways, devProject+" "+devGateway) {
		tb.Fatalf("Setup: the stack runs no gateway built from this branch, running:\n%s", gateways)
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
