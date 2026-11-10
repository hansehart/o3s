package stacktest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// AnEmptyPolicy hands the stack an old policy that holds no host, and gives the checkout its own
// policy back once the test ends.
func AnEmptyPolicy(tb testing.TB) {
	tb.Helper()
	AnOldPolicy(tb, "# empty\n")
}

// AnOldPolicy hands the stack the policy in the format the gateway read before the home's policy,
// and gives the checkout its own policy back once the test ends.
func AnOldPolicy(tb testing.TB, policy string) {
	tb.Helper()
	path := filepath.Join(Repo(tb), ".devcontainer/config.toml")
	kept, err := os.ReadFile(path)
	existed := err == nil
	// Stop when the policy exists yet reading it fails
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		tb.Fatalf("Setup: read %s: %v", path, err)
	}
	// Stop when writing the empty policy fails
	if err := os.WriteFile(path, []byte(policy), 0o644); err != nil {
		tb.Fatalf("Setup: write the old policy to %s: %v", path, err)
	}
	tb.Cleanup(func() {
		var err error
		// Restore the policy the checkout held, or remove the one the test wrote
		if existed {
			err = os.WriteFile(path, kept, 0o644)
		} else {
			err = os.Remove(path)
		}
		// Report a policy that outlives its test
		if err != nil {
			tb.Errorf("Cleanup: give %s back: %v", path, err)
		}
	})
}

// AHomePolicy hands the gateway the home's policy, mounted beside the old one. The policy and its
// mount live in the test's own folder and leave with it.
func AHomePolicy(tb testing.TB, policy string) {
	tb.Helper()
	dir := tb.TempDir()
	path := filepath.Join(dir, "o3s.json")
	// Stop when the policy stays unwritten
	if err := os.WriteFile(path, []byte(policy), 0o644); err != nil {
		tb.Fatalf("Setup: write the home's policy to %s: %v", path, err)
	}
	overlay := filepath.Join(dir, "compose.yaml")
	mount := fmt.Sprintf("services:\n  gateway:\n    volumes:\n      - %s:/config/o3s.json:ro\n", path)
	// Stop when the mount stays unwritten
	if err := os.WriteFile(overlay, []byte(mount), 0o644); err != nil {
		tb.Fatalf("Setup: write the mount to %s: %v", overlay, err)
	}
	anOverlay(tb, overlay)
}
