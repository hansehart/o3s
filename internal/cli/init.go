package cli

import (
	"fmt"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/hansehart/o3s/internal/home"
)

// newInit builds the command that makes the working folder into a new home.
func newInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Make this folder into a new o3s home",
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := workingHome()
			// Hand back a working folder that stays unknown
			if err != nil {
				return err
			}
			written, err := home.Init(dir)
			// Hand back a folder that fails to become a home
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Wrote %s\n", written[0])
			fmt.Fprintf(out, "Wrote %s%c (%d files)\n", filepath.Join(dir, ".devcontainer"), filepath.Separator, len(written)-1)
			fmt.Fprintf(out, "\nNext steps:\n")
			w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
			fmt.Fprintf(w, "  o3s firewall add <host> --port 443\tallow a host\n")
			fmt.Fprintf(w, "  echo \"$TOKEN\" | o3s secret add <NAME>\tstore a secret\n")
			fmt.Fprintf(w, "  code %s\topen it, then reopen in the container\n", dir)
			return w.Flush()
		},
	}
}
