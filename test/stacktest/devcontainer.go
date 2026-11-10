package stacktest

import (
	"encoding/json"
	"fmt"
	"testing"
)

// Raised is what the devcontainer CLI reports about the stack it raised.
type Raised struct {
	Outcome     string `json:"outcome"`
	ContainerID string `json:"containerId"`
}

// DevcontainerUp raises the daily stack the way the editor raises it, and hands back what the CLI
// reports about it.
func DevcontainerUp(tb testing.TB) (Raised, error) {
	tb.Helper()
	out, err := run(tb.Context(), tb, "devcontainer", "up", "--workspace-folder", Repo(tb))
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
