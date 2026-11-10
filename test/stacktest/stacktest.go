// Package stacktest raises parts of the o3s stack from this branch for the integration and e2e
// tests, the way httptest raises a server for HTTP tests.
//
// It drives the docker compose CLI with the daily stack and its dev overlay, which builds the
// images from this checkout, so a test exercises the code under review.
package stacktest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// compose runs docker compose on the daily stack with the dev overlay, and hands back what it printed.
func compose(ctx context.Context, tb testing.TB, args ...string) (string, error) {
	tb.Helper()
	repo := Repo(tb)
	cmd := exec.CommandContext(ctx, "docker", append([]string{
		"compose",
		"-f", filepath.Join(repo, ".devcontainer/compose.yaml"),
		"-f", filepath.Join(repo, ".devcontainer/compose.dev.yaml"),
	}, args...)...)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	// Hand back compose's own output along with the failure
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("docker compose %s: %w\n%s%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}

// Up builds the services from this branch and waits until each reports healthy. It removes them
// once the test ends.
func Up(tb testing.TB, services ...string) error {
	tb.Helper()
	anEnv(tb)
	tb.Cleanup(func() {
		// Report a service that outlives its test
		if _, err := compose(context.WithoutCancel(tb.Context()), tb, append([]string{"rm", "--stop", "--force"}, services...)...); err != nil {
			tb.Errorf("Cleanup: %v", err)
		}
	})
	_, err := compose(tb.Context(), tb, append([]string{"up", "--build", "--wait", "--wait-timeout", "60"}, services...)...)
	return err
}

// anEnv hands compose the values the host generates before every start, as its start hook does. It
// writes them when the checkout holds none, and removes what it wrote once the test ends.
func anEnv(tb testing.TB) {
	tb.Helper()
	env := filepath.Join(Repo(tb), ".devcontainer/.env")
	// Keep the values the checkout already holds
	if _, err := os.Stat(env); err == nil {
		return
	}
	values := fmt.Sprintf("O3S_MARKER=o3s-stacktest\nO3S_UID=%d\nO3S_GID=%d\n", os.Getuid(), os.Getgid())
	// Stop when writing the values fails
	if err := os.WriteFile(env, []byte(values), 0o644); err != nil {
		tb.Fatalf("Setup: write %s: %v", env, err)
	}
	tb.Cleanup(func() {
		// Report values that outlive their test
		if err := os.Remove(env); err != nil {
			tb.Errorf("Cleanup: remove %s: %v", env, err)
		}
	})
}

// Exec runs a command in a running service and hands back what it printed.
func Exec(tb testing.TB, service string, command ...string) (string, error) {
	tb.Helper()
	return compose(tb.Context(), tb, append([]string{"exec", "-T", service}, command...)...)
}

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
