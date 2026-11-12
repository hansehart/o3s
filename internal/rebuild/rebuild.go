// Package rebuild rebuilds an o3s home's cage with the devcontainer CLI, the way the editor's rebuild does.
package rebuild

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hansehart/o3s/internal/paths"
)

// CLI finds o3s's own devcontainer CLI, or one on PATH, and reports whether the host holds one.
func CLI() (string, bool) {
	installed := filepath.Join(paths.Devcontainers(), "bin", "devcontainer")
	// Take the CLI o3s installed
	if info, err := os.Stat(installed); err == nil && info.Mode()&0o111 != 0 {
		return installed, true
	}
	path, err := exec.LookPath("devcontainer")
	return path, err == nil
}

// Run rebuilds the cage of a home, passing the CLI's output on.
func Run(ctx context.Context, cli, home string, out io.Writer) error {
	cmd := exec.CommandContext(ctx, cli, "up", "--workspace-folder", home, "--remove-existing-container")
	cmd.Stdout, cmd.Stderr = out, out
	return cmd.Run()
}
