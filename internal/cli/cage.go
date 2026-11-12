package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/ide"
	"github.com/hansehart/o3s/internal/rebuild"
)

// newCage builds the commands that step into the home's cage.
func newCage() *cobra.Command {
	c := group("cage", "Open the home's cage in VS Code or a shell")
	c.AddCommand(newCageIDE(), newCageShell())
	return c
}

// newCageIDE builds the command that opens the home's workspace in VS Code straight in the cage.
func newCageIDE() *cobra.Command {
	return &cobra.Command{
		Use:   "ide",
		Short: "Open the home's workspace in VS Code inside the cage",
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := cageHome()
			// Hand back a folder that holds no home
			if err != nil {
				return err
			}
			code, err := exec.LookPath("code")
			// Point at the VS Code CLI the command runs
			if err != nil {
				return errors.New("o3s cage ide runs the VS Code CLI code, put it on PATH (VS Code: Shell Command: Install 'code' command in PATH)")
			}
			// Check that VS Code runs Docker inside WSL, where o3s runs
			if distro, ok := ide.WSL(); ok {
				settings, _ := ide.SettingsPath(cmd.Context())
				// Name the lines VS Code's settings need
				if !ide.ExecutesInWSL(settings) {
					return fmt.Errorf("VS Code runs Docker inside WSL with these lines in its settings.json:\n"+
						"  \"dev.containers.executeInWSL\": true,\n"+
						"  \"dev.containers.executeInWSLDistro\": %q", distro)
				}
			}
			host, err := ide.HostPath(cmd.Context(), dir)
			// Hand back a folder WSL fails to name
			if err != nil {
				return err
			}
			uri := ide.URI(host, filepath.Base(dir)+".code-workspace")
			// Hand back a VS Code that fails to start
			if err := exec.CommandContext(cmd.Context(), code, "--file-uri", uri).Run(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Opening %s in the cage\n", dir)
			return nil
		},
	}
}

// newCageShell builds the command that raises the home's cage and opens a shell inside it.
func newCageShell() *cobra.Command {
	return &cobra.Command{
		Use:   "shell",
		Short: "Open a shell inside the home's cage",
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := cageHome()
			// Hand back a folder that holds no home
			if err != nil {
				return err
			}
			cli, ok := rebuild.CLI()
			// Point at the devcontainer CLI o3s init installs
			if !ok {
				return errors.New("o3s cage shell runs the devcontainer CLI, install it with o3s init")
			}
			up := exec.CommandContext(cmd.Context(), cli, "up", "--workspace-folder", dir)
			up.Stdout, up.Stderr = cmd.ErrOrStderr(), cmd.ErrOrStderr()
			// Hand back a cage that fails to come up
			if err := up.Run(); err != nil {
				return fmt.Errorf("raise the cage: %w", err)
			}
			return syscall.Exec(cli, []string{cli, "exec", "--workspace-folder", dir, "bash"}, os.Environ())
		},
	}
}

// cageHome finds the home a cage command works on, and turns away a folder that holds none.
func cageHome() (string, error) {
	dir, err := workingHome()
	// Hand back a working folder that stays unknown
	if err != nil {
		return "", err
	}
	_, err = config.Load(dir)
	return dir, err
}
