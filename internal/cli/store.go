package cli

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/registry"
)

// newStore builds the commands that add, remove and list the stores the menu offers features from.
func newStore() *cobra.Command {
	s := group("store", "Add or remove the stores that offer features")
	s.AddCommand(newStoreAdd(), newStoreRm(), newStoreLs())
	return s
}

// newStoreAdd builds the command that adds a store once its collection proves it real.
func newStoreAdd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <STORE>",
		Short: "Add a store and show the features it offers",
		Args:  usage(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			var features []string
			_, err := edit(func(c *config.Config) error {
				// Turn away a store id that breaks a rule
				if _, err := c.AddStore(id); err != nil {
					return err
				}
				var err error
				features, err = registry.Collection(cmd.Context(), id)
				// Name the store whose features stay out of reach
				if err != nil {
					return fmt.Errorf("read the features of %s: %w", id, err)
				}
				return nil
			})
			// Hand back a store that fails to join the list
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n  + %s\nAdded %s\n", id, strings.Join(features, ", "), id)
			return nil
		},
	}
}

// newStoreRm builds the command that removes a store from the list.
func newStoreRm() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <STORE>",
		Short: "Remove a store",
		Args:  usage(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			_, err := edit(func(c *config.Config) error { return c.RemoveStore(id) })
			// Hand back a store that stays on the list
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", id)
			return nil
		},
	}
}

// newStoreLs builds the command that lists every store with the number of features it offers.
func newStoreLs() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List the stores and how many features each offers",
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := homeConfig()
			// Hand back a home that fails to load
			if err != nil {
				return err
			}
			counts := make([]string, len(c.Stores))
			var wg sync.WaitGroup
			// Ask every store for its collection at once
			for i, s := range c.Stores {
				wg.Go(func() {
					features, err := registry.Collection(cmd.Context(), s.ID)
					counts[i] = strconv.Itoa(len(features))
					// Warn about a store out of reach
					if err != nil {
						counts[i] = "⚠"
					}
				})
			}
			wg.Wait()
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "STORE\tFEATURES")
			// Write one row per store
			for i, s := range c.Stores {
				fmt.Fprintf(w, "%s\t%s\n", s.ID, counts[i])
			}
			return w.Flush()
		},
	}
}
