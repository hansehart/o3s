package cli

import (
	"fmt"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/devcontainer"
	"github.com/hansehart/o3s/internal/registry"
)

// newFeature builds the commands that add, remove and list the features the cage installs.
func newFeature() *cobra.Command {
	f := group("feature", "Add or remove the features the cage installs")
	f.AddCommand(newFeatureAdd(), newFeatureRm(), newFeatureLs())
	return f
}

// newFeatureAdd builds the command that adds a feature with the hosts and env it needs.
func newFeatureAdd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <id>",
		Short: "Add a feature with the hosts and env it needs",
		Args:  usage(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := registry.ParseRef(args[0])
			// Hand back an id that names no feature
			if err != nil {
				return err
			}
			s, err := new(registry.Resolver).Resolve(cmd.Context(), ref)
			// Hand back a feature its store fails to offer
			if err != nil {
				return err
			}
			pkg := s.Ref.Package()
			dir, c, dc, err := loadHome()
			// Hand back a home that fails to load
			if err != nil {
				return err
			}
			// Open every host the feature needs
			for _, h := range s.Hosts {
				// Name the host that breaks a rule, the files left as they were
				if err := c.Need(pkg, config.Change{Name: h.Name, Ports: h.Ports, Secret: h.Secret, Inject: h.Inject}); err != nil {
					return fmt.Errorf("%s needs %s: %s", pkg, h.Name, err.Error())
				}
			}
			dc.SetFeature(pkg, s.Ref.String(), s.Options)
			// Set every variable the feature brings
			for _, e := range s.Env {
				dc.SetEnv(e.Name, e.Value)
			}
			// Hand back a home that fails to save
			if err := saveHome(dir, c, dc); err != nil {
				return err
			}
			_, store, err := loadSecrets()
			// Hand back a secrets file that fails to load
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, s.Ref)
			var waiting []string
			// Show every host the feature opens, and the secret it carries
			for _, h := range s.Hosts {
				line := fmt.Sprintf("  + %s:%s", h.Name, joinPorts(registry.PortsOf(h)))
				// Name the secret, and warn while it waits for its value
				if h.Secret != "" {
					line += "  " + h.Secret
					// Warn about a secret without a value
					if _, ok := store.Value(h.Secret); !ok {
						line += " ⚠"
						waiting = append(waiting, h.Secret)
					}
				}
				fmt.Fprintln(out, line)
			}
			// Show every variable the feature sets
			for _, e := range s.Env {
				fmt.Fprintf(out, "  + %s=%s\n", e.Name, e.Value)
			}
			fmt.Fprintf(out, "Added %s\n", pkg)
			// Name how to give each waiting secret its value
			for _, name := range waiting {
				fmt.Fprintf(out, "⚠ %s waits for its value: echo \"$TOKEN\" | o3s secret add %s\n", name, name)
			}
			// Restart the stack where the allowlist changed
			if len(s.Hosts) > 0 {
				// Hand back a stack that fails to restart
				if err := restart(cmd, dir); err != nil {
					return err
				}
			}
			fmt.Fprintln(out, "Rebuild the cage to install it")
			return nil
		},
	}
}

// newFeatureRm builds the command that removes a feature with its env and the hosts it alone needs.
func newFeatureRm() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id>",
		Short: "Remove a feature with its env and the hosts it alone needs",
		Args:  usage(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := registry.ParseRef(args[0])
			// Hand back an id that names no feature
			if err != nil {
				return err
			}
			pkg := ref.Package()
			dir, c, dc, err := loadHome()
			// Hand back a home that fails to load
			if err != nil {
				return err
			}
			// Turn away a feature the home lacks
			if !dc.RemoveFeature(pkg) {
				return fmt.Errorf("%w %s", registry.ErrUnknownFeature, pkg)
			}
			s, err := new(registry.Resolver).Resolve(cmd.Context(), ref)
			// Hand back a feature whose env stays unknown, the files left as they were
			if err != nil {
				return err
			}
			ports := map[string][]int{}
			// Remember every host's ports for the hosts the removal drops
			for _, h := range c.Hosts {
				ports[h.Name] = h.Ports
			}
			dropped := c.Release(pkg)
			var removed []registry.Env
			// Drop every variable the feature set
			for _, e := range s.Env {
				// Keep the variables the config holds
				if dc.RemoveEnv(e.Name) {
					removed = append(removed, e)
				}
			}
			// Hand back a home that fails to save
			if err := saveHome(dir, c, dc); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, pkg)
			// Show every host nothing needs any more
			for _, name := range dropped {
				fmt.Fprintf(out, "  - %s:%s\n", name, joinPorts(ports[name]))
			}
			// Show every variable the removal drops
			for _, e := range removed {
				fmt.Fprintf(out, "  - %s=%s\n", e.Name, e.Value)
			}
			fmt.Fprintf(out, "Removed %s\n", pkg)
			// Restart the stack where the allowlist changed
			if len(dropped) > 0 {
				return restart(cmd, dir)
			}
			return nil
		},
	}
}

// newFeatureLs builds the command that lists every feature with its tag and the hosts it needs.
func newFeatureLs() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List the features with their version and the hosts they need",
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, c, dc, err := loadHome()
			// Hand back a home that fails to load
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "FEATURE\tVERSION\tHOSTS")
			// Write one row per feature
			for _, key := range dc.Features() {
				pkg, tag := key, "latest"
				// Split the tag off a feature a store offers
				if ref, err := registry.ParseRef(key); err == nil && ref.Tag != "" {
					pkg, tag = ref.Package(), ref.Tag
				}
				var hosts []string
				// List the hosts the feature needs
				for _, h := range c.Hosts {
					// Take a host that names the feature among its reasons
					if slices.Contains(h.By, pkg) {
						hosts = append(hosts, h.Name)
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", pkg, tag, strings.Join(hosts, ", "))
			}
			return w.Flush()
		},
	}
}

// loadHome reads the settings and the dev container config of the home the command runs in.
func loadHome() (string, config.Config, *devcontainer.Config, error) {
	dir, err := workingHome()
	// Hand back a working folder that stays unknown
	if err != nil {
		return "", config.Config{}, nil, err
	}
	c, err := config.Load(dir)
	// Hand back settings that fail to load
	if err != nil {
		return "", config.Config{}, nil, err
	}
	dc, err := devcontainer.Load(dir)
	return dir, c, dc, err
}

// saveHome writes the dev container config and then the settings of a home.
func saveHome(dir string, c config.Config, dc *devcontainer.Config) error {
	// Hand back a dev container config that fails to save
	if err := dc.Save(dir); err != nil {
		return err
	}
	return c.Save(dir)
}
