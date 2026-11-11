package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hansehart/o3s/internal/registry"
	"github.com/hansehart/o3s/internal/update"
)

// newUpdate builds the command that brings the home to the stack the binary ships and to what its stores offer.
func newUpdate() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Bring the home's stack and features up to date",
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := workingHome()
			// Hand back a working folder that stays unknown
			if err != nil {
				return err
			}
			plan, err := update.Check(cmd.Context(), dir)
			// Hand back a home that fails to compare
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			show(out, plan)
			// Leave a home that already holds what the binary and its stores ship
			if plan.Empty() {
				fmt.Fprintln(out, "Up to date")
				return nil
			}
			ok, err := confirm(cmd.InOrStdin(), out, isTerminal(cmd.InOrStdin()), yes)
			// Hand back an answer that fails to arrive
			if err != nil {
				return err
			}
			// Leave the home as it was when the user declines
			if !ok {
				fmt.Fprintln(out, "Left the home as it was")
				return nil
			}
			hosts, err := plan.Apply(dir)
			// Hand back an update that fails to write
			if err != nil {
				return err
			}
			fmt.Fprintln(out, "Updated")
			// Restart the stack where the allowlist changed
			if hosts {
				// Hand back a stack that fails to restart
				if err := restart(cmd, dir); err != nil {
					return err
				}
			}
			fmt.Fprintln(out, "Rebuild the cage to apply it")
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "apply without asking")
	return cmd
}

// show prints every change an update makes, and what it leaves for want of a store.
func show(out io.Writer, p update.Plan) {
	// Head the stack's files with their folder
	if len(p.Files) > 0 {
		fmt.Fprintln(out, ".devcontainer/")
	}
	// Show every file of the stack that changes
	for _, f := range p.Files {
		mark := "~"
		// Mark a file the home lacks
		if f.New {
			mark = "+"
		}
		fmt.Fprintf(out, "  %s %s\n", mark, f.Path)
	}
	// Show every feature that changes
	for _, fc := range p.Features {
		line := fc.Package
		// Name a tag that moves
		if fc.OldTag != fc.NewTag {
			line += fmt.Sprintf("   %s → %s", fc.OldTag, fc.NewTag)
		}
		fmt.Fprintln(out, line)
		// Show every host the feature needs anew
		for _, h := range fc.HostsAdded {
			fmt.Fprintf(out, "  + %s:%s\n", h.Name, joinPorts(registry.PortsOf(h)))
		}
		// Show every host the feature leaves
		for _, name := range fc.HostsDropped {
			fmt.Fprintf(out, "  - %s\n", name)
		}
		// Show every variable the feature brings
		for _, e := range fc.EnvAdded {
			fmt.Fprintf(out, "  + %s=%s\n", e.Name, e.Value)
		}
		// Show every variable that takes another value
		for _, e := range fc.EnvChanged {
			fmt.Fprintf(out, "  ~ %s=%s → %s\n", e.Name, e.Old, e.New)
		}
	}
	// Name every feature its store left as it was
	for _, w := range p.Warnings {
		fmt.Fprintf(out, "⚠ %s\n", w)
	}
}

// confirm asks whether to apply, takes --yes as the answer, and turns away a missing terminal.
func confirm(in io.Reader, out io.Writer, terminal, yes bool) (bool, error) {
	// Apply without asking where the user said so
	if yes {
		return true, nil
	}
	// Turn away a run where nobody can answer
	if !terminal {
		return false, fmt.Errorf("%w: pass --yes to apply without a terminal", ErrUsage)
	}
	fmt.Fprint(out, "Apply? [y/N] ")
	line, err := bufio.NewReader(in).ReadString('\n')
	// Read an answer that ends without a line break as it stands, and a failed read as a no
	if err != nil && line == "" {
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
