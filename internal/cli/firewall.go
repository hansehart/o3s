package cli

import (
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/hansehart/o3s/internal/config"
)

// newFirewall builds the commands that allow, remove and list hosts.
func newFirewall() *cobra.Command {
	fw := group("firewall", "Allow or remove the hosts the cage may reach")
	fw.AddCommand(newFirewallAdd(), newFirewallRm(), newFirewallLs())
	return fw
}

// newFirewallAdd builds the command that allows a host, or updates the one it names.
func newFirewallAdd() *cobra.Command {
	var ch config.Change
	cmd := &cobra.Command{
		Use:   "add <host>",
		Short: "Allow a host, or update it",
		Args:  usage(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			ch.Name = args[0]
			return change(func(c *config.Config) error {
				diffs, err := c.Allow(ch)
				// Hand back a change the allowlist turns away
				if err != nil {
					return err
				}
				out := cmd.OutOrStdout()
				// List the fields an update changed, under the host they belong to
				if len(diffs) > 0 {
					fmt.Fprintln(out, ch.Name)
					// Show every changed field with its value before and after
					for _, d := range diffs {
						fmt.Fprintf(out, "  ~ %s %s → %s\n", d.Field, orNone(d.Old), orNone(d.New))
					}
				}
				h, _ := c.Find(ch.Name)
				fmt.Fprintln(out, allowed(h))
				return nil
			})
		},
	}
	cmd.Flags().IntSliceVarP(&ch.Ports, "port", "p", nil, "a port to allow, 1-65535, repeatable (default 443)")
	cmd.Flags().StringVar(&ch.Secret, "secret", "", "the name of the secret the proxy injects for this host")
	cmd.Flags().StringSliceVar(&ch.Inject, "inject", nil, "where the secret goes: header, query or body, repeatable (default header)")
	return cmd
}

// newFirewallRm builds the command that takes a host the user added off the allowlist.
func newFirewallRm() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <host>",
		Short: "Remove a host",
		Args:  usage(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			return change(func(c *config.Config) error {
				kept, err := c.Remove(name)
				// Hand back a removal the allowlist turns away
				if err != nil {
					return err
				}
				// Name what still needs a host that stays
				if len(kept) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "Kept %s, still needed by %s\n", name, strings.Join(kept, ", "))
					return nil
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", name)
				return nil
			})
		},
	}
}

// newFirewallLs builds the command that lists the allowlist.
func newFirewallLs() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List the hosts",
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := home()
			// Hand back a working folder that stays unknown
			if err != nil {
				return err
			}
			c, err := config.Load(dir)
			// Hand back a home that fails to load
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "HOST\tPORTS\tSECRET\tBY")
			// Write a row per host, in the order the file holds them
			for _, h := range c.Hosts {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", h.Name, joinPorts(h.Ports), h.Secret, strings.Join(h.By, ", "))
			}
			return w.Flush()
		},
	}
}

// change loads the home, applies a change to it, and saves it when the change succeeds.
func change(apply func(*config.Config) error) error {
	dir, err := home()
	// Hand back a working folder that stays unknown
	if err != nil {
		return err
	}
	c, err := config.Load(dir)
	// Hand back a home that fails to load
	if err != nil {
		return err
	}
	// Leave the file as it was when the change fails
	if err := apply(&c); err != nil {
		return err
	}
	return c.Save(dir)
}

// allowed confirms a host with its ports, and its secret where it carries one.
func allowed(h config.Host) string {
	line := fmt.Sprintf("Allowed %s:%s", h.Name, joinPorts(h.Ports))
	// Add the secret and where it goes
	if h.Secret != "" {
		line += fmt.Sprintf(" with %s in %s", h.Secret, strings.Join(h.Inject, ", "))
	}
	return line
}

// joinPorts writes ports as one comma-separated value.
func joinPorts(ports []int) string {
	parts := make([]string, len(ports))
	// Write every port
	for i, p := range ports {
		parts[i] = strconv.Itoa(p)
	}
	return strings.Join(parts, ",")
}

// orNone shows an empty value as none.
func orNone(v string) string {
	// Name the absence of a value
	if v == "" {
		return "none"
	}
	return v
}
