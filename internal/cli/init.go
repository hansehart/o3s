package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/home"
	"github.com/hansehart/o3s/internal/host"
	"github.com/hansehart/o3s/internal/paths"
	"github.com/hansehart/o3s/internal/secrets"
)

// newInit builds the command that makes the working folder a home, and prepares the host for it on every start.
func newInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Make this folder an o3s home, or prepare the host for the home it is",
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := workingHome()
			// Hand back a working folder that stays unknown
			if err != nil {
				return err
			}
			secretsFile, err := secrets.Path()
			// Hand back a secrets file whose place stays unknown
			if err != nil {
				return err
			}
			// Turn away a home that would show the cage the host's secrets, CA or tools
			if f, ok := home.Holds(dir, filepath.Dir(secretsFile), paths.Config(), paths.State(), paths.Data()); ok {
				return fmt.Errorf("%s holds %s, which the cage would see, so o3s init needs a folder of its own, as in mkdir ~/o3s && cd ~/o3s", dir, f)
			}
			out := cmd.OutOrStdout()
			fresh := !home.Exists(dir)
			// Write a new home into a folder that holds none
			if fresh {
				written, err := home.Init(dir)
				// Hand back a folder that fails to become a home
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "Wrote %s\n", written[0])
				fmt.Fprintf(out, "Wrote %s%c (%d files)\n", filepath.Join(dir, ".devcontainer"), filepath.Separator, len(written)-1)
			}
			seeded, err := home.Seed(dir)
			// Hand back a home whose missing files stay unwritten
			if err != nil {
				return err
			}
			// Name every file the home lacked
			for _, path := range seeded {
				fmt.Fprintf(out, "Wrote %s\n", path)
			}
			// Hand back a host that fails to prepare
			if err := prepare(cmd, dir); err != nil {
				return err
			}
			// Point a new home at what comes next
			if fresh {
				nextSteps(out, dir)
				return nil
			}
			changes, err := home.Refresh(dir)
			// Point a stack that differs from this binary at update
			if err == nil && len(changes) > 0 {
				fmt.Fprintln(out, "The stack differs from this o3s, run o3s update")
			}
			return nil
		},
	}
}

// prepare readies the host for a home: the secrets file, the proxy's CA, the cage's confinement and the values compose reads.
func prepare(cmd *cobra.Command, dir string) error {
	out := cmd.OutOrStdout()
	path, err := secrets.Path()
	// Hand back a secrets file whose place stays unknown
	if err != nil {
		return err
	}
	// Hand back a secrets file that stays unwritten
	if _, err := home.SeedSecrets(path); err != nil {
		return err
	}
	caDir := paths.Proxy()
	made, err := host.CA(caDir)
	// Hand back a CA that fails to make
	if err != nil {
		return err
	}
	// Say whether the CA is new
	if made {
		fmt.Fprintf(out, "Made the proxy's CA in %s\n", caDir)
	} else {
		fmt.Fprintf(out, "The proxy's CA is ready in %s\n", caDir)
	}
	note, err := host.Profile(cmd.Context(), filepath.Join(dir, ".devcontainer", "apparmor.conf"))
	// Hand back a profile that fails to read
	if err != nil {
		return err
	}
	// Say what the confinement step found or did
	if note != "" {
		fmt.Fprintln(out, note)
	}
	// Hand back a home's state that stays unwritten
	if err := host.HomeState(dir); err != nil {
		return err
	}
	// Name every other home, since this release runs one home per machine
	for _, other := range host.OtherHomes(dir) {
		fmt.Fprintf(out, "⚠ %s is already an o3s home, and o3s runs one home per machine for now\n", other)
	}
	// Hand back values compose reads that stay unwritten
	if err := host.Env(dir); err != nil {
		return err
	}
	waiting(out, dir)
	return nil
}

// waiting names every secret a host carries that still waits for its value, with the command that gives it one.
func waiting(out io.Writer, dir string) {
	c, err := config.Load(dir)
	// Leave the names to the commands that read the settings, which report a broken file themselves
	if err != nil {
		return
	}
	_, store, err := loadSecrets()
	// Leave the names where the secrets file fails to read
	if err != nil {
		return
	}
	named := map[string]bool{}
	// Look at every host's secret
	for _, h := range c.Hosts {
		_, set := store.Value(h.Secret)
		// Name a secret once while it waits for its value
		if h.Secret != "" && !set && !named[h.Secret] {
			named[h.Secret] = true
			fmt.Fprintf(out, "⚠ %s waits for its value: echo \"$TOKEN\" | o3s secret add %s\n", h.Secret, h.Secret)
		}
	}
}

// nextSteps points a new home at the commands that fill it and the editor that opens it.
func nextSteps(out io.Writer, dir string) {
	fmt.Fprintf(out, "\nNext steps:\n")
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "  o3s firewall add <host> --port 443\tallow a host\n")
	fmt.Fprintf(w, "  echo \"$TOKEN\" | o3s secret add <NAME>\tstore a secret\n")
	fmt.Fprintf(w, "  code %s\topen it, then reopen in the container\n", filepath.Join(dir, filepath.Base(dir)+".code-workspace"))
	_ = w.Flush()
}
