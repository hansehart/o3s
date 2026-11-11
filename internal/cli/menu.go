package cli

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/menu"
)

// runMenu leads the user through a dialog and runs the command it ends in.
func runMenu(ctx context.Context, s Streams) int {
	cmd, ok, err := menu.Drive(menu.Huh{In: s.In, Out: s.Out}, menuHome())
	// Report a dialog that fails to run
	if err != nil {
		fmt.Fprintf(s.Err, "Error: %v\n", err)
		return 1
	}
	// Leave quietly when the user leaves
	if !ok {
		return 0
	}
	// Say why the dialog runs nothing
	if cmd.Note != "" {
		fmt.Fprintln(s.Out, cmd.Note)
	}
	// Leave a dialog that ends without a command
	if len(cmd.Args) == 0 {
		return 0
	}
	in := s.In
	// Hand a value the dialog collected over stdin
	if cmd.Stdin != nil {
		in = strings.NewReader(*cmd.Stdin)
	}
	return Run(ctx, cmd.Args, Streams{In: in, Out: s.Out, Err: s.Err})
}

// menuHome reads what the menu offers from the home it runs in, and as much as there is outside one.
func menuHome() menu.Home {
	dir, err := workingHome()
	// Start without a folder where the working folder stays unknown
	if err != nil {
		return menu.Home{}
	}
	h := menu.Home{Dir: dir}
	c, err := config.Load(dir)
	// Take the home's hosts and stores where it holds settings
	if err == nil {
		h.Hosts = c.Hosts
		// Offer every store by its id
		for _, s := range c.Stores {
			h.Stores = append(h.Stores, s.ID)
		}
	}
	var names []string
	// Take every secret a host carries
	for _, host := range h.Hosts {
		// Take a host's secret once
		if host.Secret != "" {
			names = append(names, host.Secret)
		}
	}
	_, store, err := loadSecrets()
	// Take every secret the secrets file names where it reads
	if err == nil {
		names = append(names, store.Names()...)
	}
	slices.Sort(names)
	// Offer every secret once, noting whether it holds a value
	for _, name := range slices.Compact(names) {
		_, set := store.Value(name)
		h.Secrets = append(h.Secrets, menu.Secret{Name: name, Set: set})
	}
	return h
}

// isInteractive reports whether both ends of a stream reach a terminal, as a menu needs.
func isInteractive(in io.Reader, out io.Writer) bool {
	return isTerminal(in) && isTerminal(out)
}
