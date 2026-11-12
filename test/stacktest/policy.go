package stacktest

import (
	"os"
	"path/filepath"
	"testing"
)

// AnEmptyPolicy hands the stack a policy without hosts for the length of the test.
func AnEmptyPolicy(tb testing.TB) {
	tb.Helper()
	AHomePolicy(tb, `{"hosts": []}`)
}

// AHomePolicy hands the stack of the test's home the policy given.
func AHomePolicy(tb testing.TB, policy string) {
	tb.Helper()
	path := filepath.Join(aHome(tb), "o3s.json")
	// Stop when the policy stays unwritten
	if err := os.WriteFile(path, []byte(policy), 0o644); err != nil {
		tb.Fatalf("Setup: write the home's policy to %s: %v", path, err)
	}
}
