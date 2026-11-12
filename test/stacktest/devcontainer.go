package stacktest

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Raised is what the devcontainer CLI reports about the stack it raised.
type Raised struct {
	Outcome     string `json:"outcome"`
	ContainerID string `json:"containerId"`
}

// DevcontainerUp raises the stack built from this branch the way the editor does.
func DevcontainerUp(tb testing.TB) (Raised, error) {
	tb.Helper()
	out, err := run(tb.Context(), tb, "devcontainer", "up", "--workspace-folder", Repo(tb), "--config", devConfig(tb))
	// Hand back the refusal the CLI reports
	if err != nil {
		return Raised{}, err
	}
	var raised Raised
	// Hand back a report the CLI printed in another shape than its result
	if err := json.Unmarshal([]byte(out), &raised); err != nil {
		return Raised{}, fmt.Errorf("read the devcontainer result %q: %w", out, err)
	}
	return raised, nil
}

// ContainerExec runs a command in a running container and hands back what it printed.
func ContainerExec(tb testing.TB, id string, command ...string) (string, error) {
	tb.Helper()
	return run(tb.Context(), tb, "docker", append([]string{"exec", id}, command...)...)
}

const (
	// composeLine is the daily config's pointer to its stack, which the dev config rewrites.
	composeLine = `"dockerComposeFile": "compose.yaml"`
	// hookLine is the daily config's start hook, which the dev config points at o3s built from this branch.
	hookLine = `"initializeCommand": "o3s init"`
)

// devConfigs holds, per test, the dev config written for it.
var devConfigs sync.Map

// devConfig hands the editor's config a stack built from this branch.
func devConfig(tb testing.TB) string {
	tb.Helper()
	// Hand back the config this test already holds, so its stack keeps one identity
	if path, ok := devConfigs.Load(tb); ok {
		return path.(string)
	}
	dir := filepath.Join(Repo(tb), ".devcontainer")
	daily, err := os.ReadFile(filepath.Join(dir, "devcontainer.json"))
	// Stop when the daily config stays unread
	if err != nil {
		tb.Fatalf("Setup: read the daily config: %v", err)
	}
	// Stop when the daily config points at its stack another way
	if !strings.Contains(string(daily), composeLine) {
		tb.Fatalf("Setup: the daily config lacks %s", composeLine)
	}
	// Stop when the daily config starts the stack another way
	if !strings.Contains(string(daily), hookLine) {
		tb.Fatalf("Setup: the daily config lacks %s", hookLine)
	}
	dev := fmt.Sprintf(`"dockerComposeFile": [%q, %q]`, filepath.Join(dir, "compose.yaml"), filepath.Join(dir, "compose.dev.yaml"))
	hook := fmt.Sprintf(`"initializeCommand": [%q, "init"]`, o3sBinary(tb))
	config := strings.Replace(strings.Replace(string(daily), composeLine, dev, 1), hookLine, hook, 1)
	path := filepath.Join(tb.TempDir(), "devcontainer.json")
	// Stop when the dev config stays unwritten
	if err := os.WriteFile(path, []byte(config), 0o644); err != nil {
		tb.Fatalf("Setup: write the dev config: %v", err)
	}
	devConfigs.Store(tb, path)
	tb.Cleanup(func() { devConfigs.Delete(tb) })
	return path
}

// o3sBinary builds o3s from this branch for a test, so the stack starts through the hook under review.
func o3sBinary(tb testing.TB) string {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "o3s")
	cmd := exec.CommandContext(tb.Context(), "go", "build", "-o", path, "./cmd/o3s")
	cmd.Dir = Repo(tb)
	out, err := cmd.CombinedOutput()
	// Stop when the binary fails to build
	if err != nil {
		tb.Fatalf("Setup: build o3s: %v\n%s", err, out)
	}
	return path
}
