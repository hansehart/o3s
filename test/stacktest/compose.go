package stacktest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Repo finds the root of this checkout, the directory that holds the module definition.
func Repo(tb testing.TB) string {
	tb.Helper()
	dir, err := os.Getwd()
	// Stop when the working directory is unreadable
	if err != nil {
		tb.Fatalf("Setup: read the working directory: %v", err)
	}
	// Walk up from the working directory to the root of the checkout
	for {
		// Hand back the first directory that holds the module definition
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		// Stop once the walk reaches the filesystem root
		if parent == dir {
			tb.Fatalf("Setup: find go.mod above the working directory")
		}
		dir = parent
	}
}

// compose runs docker compose on the stack of the test's home with the dev overlay, and hands back what it printed.
func compose(ctx context.Context, tb testing.TB, args ...string) (string, error) {
	tb.Helper()
	files := []string{
		"compose",
		"-f", filepath.Join(aHome(tb), ".devcontainer", "compose.yaml"),
		"-f", filepath.Join(Repo(tb), ".devcontainer", "compose.dev.yaml"),
	}
	return run(ctx, tb, "docker", append(files, args...)...)
}

// Up builds the services from this branch in the test's home and waits until they are healthy. It removes them once
// the test ends.
func Up(tb testing.TB, services ...string) error {
	tb.Helper()
	tb.Cleanup(func() {
		// Report a service that outlives its test
		if _, err := compose(context.WithoutCancel(tb.Context()), tb, append([]string{"rm", "--stop", "--force"}, services...)...); err != nil {
			tb.Errorf("Cleanup: %v", err)
		}
	})
	_, err := compose(tb.Context(), tb, append([]string{"up", "--build", "--wait", "--wait-timeout", "60"}, services...)...)
	return err
}

// Exec runs a command in a running service and hands back what it printed.
func Exec(tb testing.TB, service string, command ...string) (string, error) {
	tb.Helper()
	return compose(tb.Context(), tb, append([]string{"exec", "-T", service}, command...)...)
}
