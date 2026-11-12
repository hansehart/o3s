package stacktest

import (
	"encoding/json"
	"fmt"
	"os"
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

// DevcontainerUp raises the stack of the test's home, built from this branch, the way the editor does.
func DevcontainerUp(tb testing.TB) (Raised, error) {
	tb.Helper()
	out, err := run(tb.Context(), tb, "devcontainer", "up", "--workspace-folder", devHome(tb))
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

// composeLine is a home's pointer to its stack, which the dev overlay joins.
const composeLine = `"dockerComposeFile": "compose.yaml"`

// overlaid holds, per test, whether its home's config names the dev overlay already.
var overlaid sync.Map

// devHome hands back the test's home with its config joining the dev overlay, so every image comes from this branch.
func devHome(tb testing.TB) string {
	tb.Helper()
	dir := aHome(tb)
	// Hand back a home whose config names the overlay already
	if _, ok := overlaid.Load(tb); ok {
		return dir
	}
	path := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	config, err := os.ReadFile(path)
	// Stop when the home's config stays unread
	if err != nil {
		tb.Fatalf("Setup: read the home's config: %v", err)
	}
	// Stop when the home's config points at its stack another way
	if !strings.Contains(string(config), composeLine) {
		tb.Fatalf("Setup: the home's config lacks %s", composeLine)
	}
	dev := fmt.Sprintf(`"dockerComposeFile": ["compose.yaml", %q]`, filepath.Join(Repo(tb), "test", "stacktest", "compose.dev.yaml"))
	// Stop when the config stays unwritten
	if err := os.WriteFile(path, []byte(strings.Replace(string(config), composeLine, dev, 1)), 0o644); err != nil {
		tb.Fatalf("Setup: write the home's config: %v", err)
	}
	overlaid.Store(tb, true)
	tb.Cleanup(func() { overlaid.Delete(tb) })
	return dir
}
