// Package rebuild rebuilds an o3s home's cage with the devcontainer CLI, the way the editor's rebuild does.
package rebuild

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// CLI finds the devcontainer CLI in the folder its installer uses, or on PATH, and reports whether the host holds one.
func CLI() (string, bool) {
	prefix := os.Getenv("DEVCONTAINERS_INSTALL_DIR")
	// Look in the installer's default folder where the environment names none
	if prefix == "" {
		home, err := os.UserHomeDir()
		// Look on PATH alone where the home folder stays unknown
		if err == nil {
			prefix = filepath.Join(home, ".devcontainers")
		}
	}
	installed := filepath.Join(prefix, "bin", "devcontainer")
	// Take the CLI the installer put in place
	if info, err := os.Stat(installed); prefix != "" && err == nil && info.Mode()&0o111 != 0 { //nolint:gosec // the folder is the user's own, as their installer reads it
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
