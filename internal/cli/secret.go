package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/secrets"
)

// newSecret builds the commands that store, remove and list the values the proxy injects.
func newSecret() *cobra.Command {
	s := group("secret", "Store or remove the values the proxy injects")
	s.AddCommand(newSecretAdd(), newSecretRm(), newSecretLs())
	return s
}

// newSecretAdd builds the command that stores a secret's value, read from stdin.
func newSecretAdd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <NAME>",
		Short: "Store a secret's value, read from stdin",
		Args:  usage(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			in := cmd.InOrStdin()
			// Turn away a value typed at a terminal, so it stays out of the screen and the history
			if isTerminal(in) {
				return fmt.Errorf("%w: pipe the value on stdin, as in: echo \"$TOKEN\" | o3s secret add %s", ErrUsage, name)
			}
			value, err := io.ReadAll(in)
			// Hand back a stdin that fails to read
			if err != nil {
				return err
			}
			path, store, err := loadSecrets()
			// Hand back a secrets file that fails to load
			if err != nil {
				return err
			}
			replaced, err := store.Set(name, string(value))
			// Hand back a value the proxy would refuse
			if err != nil {
				return err
			}
			// Hand back a secrets file that fails to save
			if err := store.Save(path); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			// Report a rotation without the value
			if replaced {
				fmt.Fprintf(out, "%s\n  ~ value replaced\n", name)
			}
			fmt.Fprintf(out, "Stored %s\n", name)
			// Point a name no host carries at the command that binds it
			if len(hostsNaming(name)) == 0 {
				fmt.Fprintf(out, "Bind it to a host with: o3s firewall add <host> --secret %s\n", name)
			}
			return nil
		},
	}
}

// newSecretRm builds the command that removes a secret's value.
func newSecretRm() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <NAME>",
		Short: "Remove a secret's value",
		Args:  usage(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			path, store, err := loadSecrets()
			// Hand back a secrets file that fails to load
			if err != nil {
				return err
			}
			// Turn away a name the file lacks
			if !store.Remove(name) {
				return fmt.Errorf("%w %s", secrets.ErrUnknown, name)
			}
			// Hand back a secrets file that fails to save
			if err := store.Save(path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", name)
			return nil
		},
	}
}

// newSecretLs builds the command that lists every secret, whether it is set, and its hosts.
func newSecretLs() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List the secrets",
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, store, err := loadSecrets()
			// Hand back a secrets file that fails to load
			if err != nil {
				return err
			}
			names := store.Names()
			c, _ := homeConfig()
			// Add the names the home's hosts carry, stored or not
			for _, h := range c.Hosts {
				// Add a name the file is yet to hold
				if h.Secret != "" && !slices.Contains(names, h.Secret) {
					names = append(names, h.Secret)
				}
			}
			slices.Sort(names)
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tVALUE\tHOSTS")
			// Write a row per name
			for _, name := range names {
				state := "empty"
				// Show that a value is stored, never the value
				if _, ok := store.Value(name); ok {
					state = "set"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", name, state, strings.Join(hostsNaming(name), ", "))
			}
			return w.Flush()
		},
	}
}

// loadSecrets finds and reads the secrets file the stack mounts into the proxy.
func loadSecrets() (string, secrets.Store, error) {
	path, err := secrets.Path()
	// Hand back a secrets file whose place stays unknown
	if err != nil {
		return "", secrets.Store{}, err
	}
	store, err := secrets.Load(path)
	return path, store, err
}

// homeConfig reads the settings of the home the command runs in.
func homeConfig() (config.Config, error) {
	dir, err := workingHome()
	// Hand back a working folder that stays unknown
	if err != nil {
		return config.Config{}, err
	}
	return config.Load(dir)
}

// hostsNaming lists the hosts of the home that carry a secret, and none outside a home.
func hostsNaming(name string) []string {
	c, err := homeConfig()
	// Read no hosts outside a home
	if err != nil && errors.Is(err, config.ErrNoHome) {
		return nil
	}
	var hosts []string
	// Collect every host that carries the name
	for _, h := range c.Hosts {
		// Keep a host that carries the name
		if h.Secret == name {
			hosts = append(hosts, h.Name)
		}
	}
	return hosts
}

// isTerminal reports whether a stream reaches a terminal rather than a pipe or a file.
func isTerminal(stream any) bool {
	f, ok := stream.(*os.File)
	// Read anything but a file as a pipe
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
