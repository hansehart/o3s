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
	"github.com/hansehart/o3s/internal/secrets"
)

// newFeature builds the commands that add, remove and list the features the cage installs.
func newFeature() *cobra.Command {
	f := group("feature", "Add or remove the features the cage installs")
	f.AddCommand(newFeatureAdd(), newFeatureRm(), newFeatureLs())
	return f
}

// newFeatureAdd builds the command that adds a feature with the hosts and env it needs.
func newFeatureAdd() *cobra.Command {
	var noPreview bool
	cmd := &cobra.Command{
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
			lines, waiting := addLines(s, store)
			// Show what the feature brings, unless the menu showed it already
			if !noPreview {
				// Print every line of the change
				for _, line := range lines {
					fmt.Fprintln(out, line)
				}
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
	previewFlag(cmd, &noPreview)
	return cmd
}

// newFeatureRm builds the command that removes a feature with its env and the hosts it alone needs.
func newFeatureRm() *cobra.Command {
	var noPreview bool
	cmd := &cobra.Command{
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
			r := removal(c, dc, s)
			c.Release(pkg)
			// Drop every variable the feature set
			for _, e := range r.env {
				dc.RemoveEnv(e.Name)
			}
			// Hand back a home that fails to save
			if err := saveHome(dir, c, dc); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			// Show what leaves with the feature, unless the menu showed it already
			if !noPreview {
				// Print every line of the change
				for _, line := range rmLines(pkg, r) {
					fmt.Fprintln(out, line)
				}
			}
			fmt.Fprintf(out, "Removed %s\n", pkg)
			// Restart the stack where the allowlist changed
			if len(r.hosts) > 0 {
				return restart(cmd, dir)
			}
			return nil
		},
	}
	previewFlag(cmd, &noPreview)
	return cmd
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

// addLines shows what a feature brings, and lists the secrets it waits for.
func addLines(s registry.Setup, store secrets.Store) ([]string, []string) {
	lines := []string{s.Ref.String()}
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
		lines = append(lines, line)
	}
	// Show every variable the feature sets
	for _, e := range s.Env {
		lines = append(lines, fmt.Sprintf("  + %s=%s", e.Name, e.Value))
	}
	return lines, waiting
}

// removalOf is what removing a feature takes from a home: the hosts nothing else needs and the env it set.
type removalOf struct {
	hosts []string
	env   []registry.Env
}

// removal works out what removing a feature takes from a home, leaving the home as it is.
func removal(c config.Config, dc *devcontainer.Config, s registry.Setup) removalOf {
	ports := map[string][]int{}
	// Remember every host's ports for the hosts the removal drops
	for _, h := range c.Hosts {
		ports[h.Name] = h.Ports
	}
	after := c
	after.Hosts = slices.Clone(c.Hosts)
	var r removalOf
	// Name every host nothing else needs, with its ports
	for _, name := range after.Release(s.Ref.Package()) {
		r.hosts = append(r.hosts, name+":"+joinPorts(ports[name]))
	}
	// Take every variable of the feature the config sets
	for _, e := range s.Env {
		// Take a variable the config holds
		if _, ok := dc.Env(e.Name); ok {
			r.env = append(r.env, e)
		}
	}
	return r
}

// rmLines shows what leaves with a feature.
func rmLines(pkg string, r removalOf) []string {
	lines := []string{pkg}
	// Show every host nothing needs any more
	for _, h := range r.hosts {
		lines = append(lines, "  - "+h)
	}
	// Show every variable the removal drops
	for _, e := range r.env {
		lines = append(lines, fmt.Sprintf("  - %s=%s", e.Name, e.Value))
	}
	return lines
}

// previewFlag adds the hidden flag the menu passes once it has shown a feature's change.
func previewFlag(cmd *cobra.Command, noPreview *bool) {
	cmd.Flags().BoolVar(noPreview, "no-preview", false, "leave out the change the menu showed already")
	_ = cmd.Flags().MarkHidden("no-preview")
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
