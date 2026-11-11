package cli

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/devcontainer"
	"github.com/hansehart/o3s/internal/menu"
	"github.com/hansehart/o3s/internal/registry"
)

// runMenu leads the user through a dialog and runs the commands it ends in, in order.
func runMenu(ctx context.Context, s Streams) int {
	cmds, ok, err := menu.Drive(menu.Huh{In: s.In, Out: s.Out}, menuHome(ctx))
	// Report a dialog that fails to run
	if err != nil {
		fmt.Fprintf(s.Err, "Error: %v\n", err)
		return 1
	}
	// Leave quietly when the user leaves
	if !ok {
		return 0
	}
	// Run every command, stopping at the first that fails
	for _, cmd := range cmds {
		// Say why the dialog runs nothing
		if cmd.Note != "" {
			fmt.Fprintln(s.Out, cmd.Note)
		}
		// Pass over a command that only carries a note
		if len(cmd.Args) == 0 {
			continue
		}
		in := s.In
		// Hand a value the dialog collected over stdin
		if cmd.Stdin != nil {
			in = strings.NewReader(*cmd.Stdin)
		}
		// Stop at a command that fails
		if code := Run(ctx, cmd.Args, Streams{In: in, Out: s.Out, Err: s.Err}); code != 0 {
			return code
		}
	}
	return 0
}

// menuLookup answers the menu's questions to the stores, reading each store once.
type menuLookup struct {
	// ctx is the run's context, which the menu's questions ask the stores within.
	ctx context.Context
	r   *registry.Resolver
}

// Features lists the ids of the features a store offers.
func (l menuLookup) Features(store string) ([]string, error) {
	features, err := l.r.Features(l.ctx, store)
	// Name the store whose features stay out of reach
	if err != nil {
		return nil, fmt.Errorf("read the features of %s: %w", store, err)
	}
	return ids(features), nil
}

// Add shows what adding a feature brings, as feature add prints it.
func (l menuLookup) Add(id string) (menu.Preview, error) {
	ref, err := registry.ParseRef(id)
	// Hand back an id that names no feature
	if err != nil {
		return menu.Preview{}, err
	}
	s, err := l.r.Resolve(l.ctx, ref)
	// Hand back a feature its store fails to offer
	if err != nil {
		return menu.Preview{}, err
	}
	_, store, err := loadSecrets()
	// Hand back a secrets file that fails to load
	if err != nil {
		return menu.Preview{}, err
	}
	lines, waiting := addLines(s, store)
	return menu.Preview{Lines: lines, Waiting: waiting}, nil
}

// Remove shows what leaves with a feature, as feature rm prints it.
func (l menuLookup) Remove(pkg string) (menu.Preview, error) {
	ref, err := registry.ParseRef(pkg)
	// Hand back an id that names no feature
	if err != nil {
		return menu.Preview{}, err
	}
	s, err := l.r.Resolve(l.ctx, ref)
	// Hand back a feature whose env stays unknown
	if err != nil {
		return menu.Preview{}, err
	}
	_, c, dc, err := loadHome()
	// Hand back a home that fails to load
	if err != nil {
		return menu.Preview{}, err
	}
	return menu.Preview{Lines: rmLines(pkg, removal(c, dc, s))}, nil
}

// menuHome reads what the menu offers from the home it runs in, and as much as there is outside one.
func menuHome(ctx context.Context) menu.Home {
	lookup := menuLookup{ctx: ctx, r: &registry.Resolver{}}
	dir, err := workingHome()
	// Start without a folder where the working folder stays unknown
	if err != nil {
		return menu.Home{Lookup: lookup}
	}
	h := menu.Home{Dir: dir, Lookup: lookup}
	c, err := config.Load(dir)
	// Take the home's hosts and stores where it holds settings
	if err == nil {
		h.Hosts = c.Hosts
		// Offer every store by its id
		for _, s := range c.Stores {
			h.Stores = append(h.Stores, s.ID)
		}
	}
	dc, err := devcontainer.Load(dir)
	// Offer every feature the home installs from a store
	if err == nil {
		// Take every feature that names a store
		for _, key := range dc.Features() {
			// Offer the feature by its package
			if ref, err := registry.ParseRef(key); err == nil {
				h.Features = append(h.Features, ref.Package())
			}
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
