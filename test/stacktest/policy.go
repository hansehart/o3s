package stacktest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// AnEmptyPolicy hands the stack a policy that holds no host, and gives the checkout its own policy
// back once the test ends.
func AnEmptyPolicy(tb testing.TB) {
	tb.Helper()
	policy := filepath.Join(Repo(tb), ".devcontainer/config.toml")
	kept, err := os.ReadFile(policy)
	existed := err == nil
	// Stop when the policy exists yet reading it fails
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		tb.Fatalf("Setup: read %s: %v", policy, err)
	}
	// Stop when writing the empty policy fails
	if err := os.WriteFile(policy, []byte("# empty\n"), 0o644); err != nil {
		tb.Fatalf("Setup: write an empty policy to %s: %v", policy, err)
	}
	tb.Cleanup(func() {
		var err error
		// Restore the policy the checkout held, or remove the one the test wrote
		if existed {
			err = os.WriteFile(policy, kept, 0o644)
		} else {
			err = os.Remove(policy)
		}
		// Report a policy that outlives its test
		if err != nil {
			tb.Errorf("Cleanup: give %s back: %v", policy, err)
		}
	})
}
