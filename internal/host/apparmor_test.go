package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// confinement is a host the test plays: whether AppArmor runs, whether the kernel holds the profile, and how sudo answers.
type confinement struct {
	enabled, loaded, installedSame bool
	sudo                           string
}

// aHostWith sets up a host as the case describes and hands back the profile file and sudo's log.
func aHostWith(t *testing.T, c confinement) (string, string) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	sec := filepath.Join(dir, "sec")
	profile := filepath.Join(dir, "apparmor.conf")
	log := filepath.Join(dir, "sudo.log")
	enabled := "exit 1"
	// Answer as a host whose kernel runs AppArmor
	if c.enabled {
		enabled = "exit 0"
	}
	fakes := map[string]string{
		"aa-enabled": enabled + "\n",
		"sudo":       "echo \"$*\" >> " + log + "\n" + c.sudo + "\n",
	}
	// Write every fake command
	for name, body := range fakes {
		// Stop when a fake stays unwritten
		if err := os.MkdirAll(bin, 0o750); err != nil {
			t.Fatalf("Setup: %v", err)
		}
		// Stop when a fake stays unwritten
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil { //nolint:gosec // the test runs its fakes
			t.Fatalf("Setup: %v", err)
		}
	}
	t.Setenv("PATH", bin+":/usr/bin:/bin")
	// Stop when the profile stays unwritten
	if err := os.WriteFile(profile, []byte("profile o3s-cage {}\n"), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Lay out the kernel's policy as a host that loaded the profile
	if c.loaded {
		// Stop when the policy stays unwritten
		if err := os.MkdirAll(filepath.Join(sec, "apparmor", "policy", "profiles", "o3s-cage.42"), 0o750); err != nil {
			t.Fatalf("Setup: %v", err)
		}
	}
	mounts := filepath.Join(dir, "mounts")
	// Stop when the mount table stays unwritten
	if err := os.WriteFile(mounts, []byte("proc /proc proc rw 0 0\nsecurityfs "+sec+" securityfs rw 0 0\n"), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	installedProfile := filepath.Join(dir, "etc-apparmor.d-o3s-cage")
	// Leave the profile where the host loads it at boot, the same as the stack's or another
	if c.installedSame {
		// Stop when the installed profile stays unwritten
		if err := os.WriteFile(installedProfile, []byte("profile o3s-cage {}\n"), 0o600); err != nil {
			t.Fatalf("Setup: %v", err)
		}
	}
	oldMounts, oldInstalled := mountTable, installed
	mountTable, installed = mounts, installedProfile
	t.Cleanup(func() { mountTable, installed = oldMounts, oldInstalled })
	return profile, log
}

// The profile gets loaded where the kernel lacks it, kept where it holds it, and left alone without AppArmor.
func TestProfileLoadsWhatTheKernelLacks(t *testing.T) {
	tests := []struct {
		name    string
		host    confinement
		want    string
		sudoRan bool
	}{
		{"a host without AppArmor", confinement{}, "", false},
		{"a host that holds the profile", confinement{enabled: true, loaded: true, installedSame: true}, "the cage's confinement profile is loaded", false},
		{"a host that lost the profile", confinement{enabled: true, installedSame: true, sudo: "exit 0"}, "loaded the cage's confinement profile", true},
		{"a host that withholds sudo", confinement{enabled: true, sudo: "exit 1"}, "load this host's confinement profile: sudo apparmor_parser -r ", true},
	}
	// Check every host
	for _, tt := range tests {
		profile, log := aHostWith(t, tt.host)

		got, err := Profile(t.Context(), profile)

		// Report a message that differs
		if err != nil || !strings.HasPrefix(got, tt.want) || (tt.want == "" && got != "") {
			t.Errorf("%s: Profile() = %q, %v, want %q", tt.name, got, err, tt.want)
		}
		calls, _ := os.ReadFile(log)
		// Require sudo asked exactly where the kernel lacks the profile
		if (len(calls) > 0) != tt.sudoRan {
			t.Errorf("%s: sudo calls %q, want ran %v", tt.name, calls, tt.sudoRan)
		}
		// Require the load and the install sudo runs
		if tt.sudoRan && !strings.Contains(string(calls), "apparmor_parser -r -T") {
			t.Errorf("%s: sudo calls %q lack the parser", tt.name, calls)
		}
	}
}
