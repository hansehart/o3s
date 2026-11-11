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
	AHomePolicy(tb, `{"hosts": []}`)
}

// AHomePolicy hands the stack the home's policy, and gives the checkout its own policy back once
// the test ends.
func AHomePolicy(tb testing.TB, policy string) {
	tb.Helper()
	path := filepath.Join(Repo(tb), "o3s.json")
	kept, err := os.ReadFile(path)
	existed := err == nil
	// Stop when the policy exists yet reading it fails
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		tb.Fatalf("Setup: read %s: %v", path, err)
	}
	// Stop when the policy stays unwritten
	if err := os.WriteFile(path, []byte(policy), 0o644); err != nil {
		tb.Fatalf("Setup: write the home's policy to %s: %v", path, err)
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
