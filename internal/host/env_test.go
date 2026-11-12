package host

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The values compose reads name this user and keep the marker the cage already holds.
func TestEnvKeepsTheMarkerAndNamesTheUser(t *testing.T) {
	home := t.TempDir()
	// Stop when the stack folder fails to appear
	if err := os.MkdirAll(filepath.Join(home, ".devcontainer"), 0o750); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Stop when the first run fails
	if err := Env(home); err != nil {
		t.Fatalf("Env() = %v", err)
	}
	first, _ := os.ReadFile(filepath.Join(home, ".devcontainer", ".env"))

	err := Env(home)

	// Stop when the second run fails
	if err != nil {
		t.Fatalf("Env() again = %v", err)
	}
	second, _ := os.ReadFile(filepath.Join(home, ".devcontainer", ".env"))
	marker := regexp.MustCompile(`(?m)^O3S_MARKER=o3s-[0-9a-f]{64}$`).Find(first)
	// Require a fresh marker on the first run, kept on the second
	if marker == nil || !strings.Contains(string(second), string(marker)) {
		t.Errorf("markers %q then:\n%s", marker, second)
	}
	// Require this user's ids
	for _, want := range []string{fmt.Sprintf("O3S_UID=%d\n", os.Getuid()), fmt.Sprintf("O3S_GID=%d\n", os.Getgid())} {
		// Report an id the file lacks
		if !strings.Contains(string(second), want) {
			t.Errorf(".env lacks %q:\n%s", want, second)
		}
	}
}
