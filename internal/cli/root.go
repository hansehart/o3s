package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newRoot builds the o3s command tree, wired to the given streams.
func newRoot(s Streams) *cobra.Command {
	root := &cobra.Command{
		Use:           "o3s",
		Short:         "Set up the cage an o3s home runs",
		Args:          usage(cobra.NoArgs),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.SetIn(s.In)
	root.SetOut(s.Out)
	root.SetErr(s.Err)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	})
	root.AddCommand(newFirewall(), newSecret())
	return root
}

// usage marks every rejected argument list as a usage error.
func usage(args cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, a []string) error {
		// Mark the rejection as a usage error
		if err := args(cmd, a); err != nil {
			return fmt.Errorf("%w: %w", ErrUsage, err)
		}
		return nil
	}
}

// group builds a command that only holds actions, so naming none is a usage error.
func group(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  usage(cobra.NoArgs),
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("%w: name an action", ErrUsage)
		},
	}
}

// home finds the o3s home a command works on, the folder it runs in.
func home() (string, error) {
	return os.Getwd()
}
