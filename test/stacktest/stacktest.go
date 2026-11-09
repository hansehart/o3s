// Package stacktest raises parts of the o3s stack from this branch for the integration and e2e
// tests, the way httptest raises a server for HTTP tests.
//
// It drives the docker compose CLI with the daily stack and its dev overlay, which builds the
// images from this checkout, so a test exercises the code under review.
package stacktest

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Repo finds the root of this checkout, the directory that holds go.mod.
func Repo(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Setup: read the working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("Setup: find go.mod above the working directory")
		}
		dir = parent
	}
}

// compose runs docker compose on the daily stack with the dev overlay, and hands back what it printed.
func compose(t testing.TB, args ...string) (string, error) {
	t.Helper()
	repo := Repo(t)
	cmd := exec.Command("docker", append([]string{
		"compose",
		"-f", filepath.Join(repo, ".devcontainer/compose.yaml"),
		"-f", filepath.Join(repo, ".devcontainer/compose.dev.yaml"),
	}, args...)...)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("docker compose %s: %w\n%s%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}

// Up builds the services from this branch and waits until each reports healthy. It removes them
// once the test ends.
func Up(t testing.TB, services ...string) error {
	t.Helper()
	anEnv(t)
	t.Cleanup(func() {
		if _, err := compose(t, append([]string{"rm", "--stop", "--force"}, services...)...); err != nil {
			t.Errorf("Cleanup: %v", err)
		}
	})
	_, err := compose(t, append([]string{"up", "--build", "--wait", "--wait-timeout", "60"}, services...)...)
	return err
}

// anEnv hands compose the values the host generates before every start, as initialize.sh does. It
// writes them when the checkout holds none, and removes what it wrote once the test ends.
func anEnv(t testing.TB) {
	t.Helper()
	env := filepath.Join(Repo(t), ".devcontainer/.env")
	if _, err := os.Stat(env); err == nil {
		return
	}
	values := fmt.Sprintf("O3S_MARKER=o3s-stacktest\nO3S_UID=%d\nO3S_GID=%d\n", os.Getuid(), os.Getgid())
	if err := os.WriteFile(env, []byte(values), 0o644); err != nil {
		t.Fatalf("Setup: write %s: %v", env, err)
	}
	t.Cleanup(func() {
		if err := os.Remove(env); err != nil {
			t.Errorf("Cleanup: remove %s: %v", env, err)
		}
	})
}

// Exec runs a command in a running service and hands back what it printed.
func Exec(t testing.TB, service string, command ...string) (string, error) {
	t.Helper()
	return compose(t, append([]string{"exec", "-T", service}, command...)...)
}

// AnEmptyPolicy hands the stack a policy that holds no host, and gives the checkout its own policy
// back once the test ends.
func AnEmptyPolicy(t testing.TB) {
	t.Helper()
	policy := filepath.Join(Repo(t), ".devcontainer/config.toml")
	kept, err := os.ReadFile(policy)
	existed := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Setup: read %s: %v", policy, err)
	}
	if err := os.WriteFile(policy, []byte("# empty\n"), 0o644); err != nil {
		t.Fatalf("Setup: write an empty policy to %s: %v", policy, err)
	}
	t.Cleanup(func() {
		var err error
		if existed {
			err = os.WriteFile(policy, kept, 0o644)
		} else {
			err = os.Remove(policy)
		}
		if err != nil {
			t.Errorf("Cleanup: give %s back: %v", policy, err)
		}
	})
}
