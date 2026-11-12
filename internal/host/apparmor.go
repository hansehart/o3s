package host

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// profileName is the name the cage's confinement profile carries.
const profileName = "o3s-cage"

var (
	// mountTable is where the kernel lists its mounts.
	mountTable = "/proc/mounts"
	// installed is where the host keeps the profile it loads at every boot.
	installed = "/etc/apparmor.d/" + profileName
)

// Profile loads the cage's confinement profile where the kernel runs AppArmor and lacks it, and says what it found or did.
func Profile(ctx context.Context, file string) (string, error) {
	// Leave a host without AppArmor alone
	if exec.CommandContext(ctx, "aa-enabled", "--quiet").Run() != nil {
		return "", nil
	}
	want, err := os.ReadFile(file)
	// Hand back a profile that fails to read
	if err != nil {
		return "", err
	}
	have, _ := os.ReadFile(installed)
	// Leave the kernel alone where it holds the profile this host installed
	if loaded() && bytes.Equal(want, have) {
		return "the cage's confinement profile is loaded", nil
	}
	cmd := exec.CommandContext(ctx, "sudo", "sh", "-c", `apparmor_parser -r -T "$1" && install -m 644 "$1" "$2"`, "_", file, installed)
	// Load it and leave it where the host loads it at every boot, or name the step for whoever holds the privilege
	if cmd.Run() != nil {
		return "load this host's confinement profile: sudo apparmor_parser -r " + file + " && sudo install -m 644 " + file + " " + installed, nil
	}
	return "loaded the cage's confinement profile", nil
}

// loaded reports whether the kernel's policy names the cage's profile.
func loaded() bool {
	data, err := os.ReadFile(mountTable)
	// Read a mount table that fails to read as no policy
	if err != nil {
		return false
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	// Look at every mount for the one that carries the policy
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		// Pass over every mount but the security file system
		if len(fields) < 3 || fields[2] != "securityfs" {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(fields[1], "apparmor", "policy", "profiles", profileName+".*"))
		// Report a policy that holds the profile
		if len(matches) > 0 {
			return true
		}
	}
	return false
}
