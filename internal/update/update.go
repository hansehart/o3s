// Package update brings an o3s home to the stack the binary ships and to what its features' stores
// now offer.
package update

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/devcontainer"
	"github.com/hansehart/o3s/internal/home"
	"github.com/hansehart/o3s/internal/registry"
)

// Plan is everything an update changes in a home, and what it leaves for want of a store.
type Plan struct {
	Files    []home.FileChange
	Features []FeatureChange
	Warnings []string
}

// FeatureChange is what an update changes for one feature.
type FeatureChange struct {
	Package      string
	OldTag       string
	NewTag       string
	HostsAdded   []registry.Host
	HostsDropped []string
	EnvAdded     []registry.Env
	EnvChanged   []EnvChange
	setup        registry.Setup
	hosts        bool
}

// EnvChange is a variable an update gives another value.
type EnvChange struct {
	Name, Old, New string
}

// Empty reports whether the home already holds what the binary and the stores ship.
func (p Plan) Empty() bool {
	return len(p.Files) == 0 && len(p.Features) == 0
}

// Check works out what an update changes in a home, asking each store once.
func Check(ctx context.Context, dir string) (Plan, error) {
	files, err := home.Refresh(dir)
	// Hand back a stack that fails to compare
	if err != nil {
		return Plan{}, err
	}
	c, err := config.Load(dir)
	// Hand back settings that fail to load
	if err != nil {
		return Plan{}, err
	}
	dc, err := devcontainer.Load(dir)
	// Hand back a config that fails to load
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Files: files}
	var r registry.Resolver
	// Check every feature a store offers
	for _, key := range dc.Features() {
		ref, err := registry.ParseRef(key)
		// Leave a feature that comes from no store
		if err != nil {
			continue
		}
		tag := ref.Tag
		ref.Tag = ""
		s, err := r.Resolve(ctx, ref)
		// Name a feature whose store fails to tell what it now needs
		if err != nil {
			p.Warnings = append(p.Warnings, fmt.Sprintf("%s keeps its version: %v", ref.Package(), err))
			continue
		}
		// Keep a tag that pins more than a major version or follows the newest release
		if !isMajor(tag) {
			s.Ref.Tag = tag
		}
		fc := change(c, dc, tag, s)
		// Name a feature that changes
		if fc.OldTag != fc.NewTag || fc.hosts || len(fc.EnvAdded)+len(fc.EnvChanged) > 0 {
			p.Features = append(p.Features, fc)
		}
	}
	return p, nil
}

// change works out what a feature's update changes in the settings and the config.
func change(c config.Config, dc *devcontainer.Config, tag string, s registry.Setup) FeatureChange {
	pkg := s.Ref.Package()
	fc := FeatureChange{Package: pkg, OldTag: tag, NewTag: s.Ref.Tag, setup: s}
	before := neededBy(c, pkg)
	after := c
	after.Hosts = slices.Clone(c.Hosts)
	dropped := after.Release(pkg)
	// Open every host the feature now needs, leaving a host that breaks a rule to the apply
	for _, h := range s.Hosts {
		_ = after.Need(pkg, config.Change{Name: h.Name, Ports: h.Ports, Secret: h.Secret, Inject: h.Inject})
		// Name a host the feature needs anew
		if !slices.Contains(before, h.Name) {
			fc.HostsAdded = append(fc.HostsAdded, h)
		}
	}
	// Name every host the feature leaves and nothing else needs
	for _, name := range dropped {
		// Take a host the feature no longer needs
		if !slices.ContainsFunc(s.Hosts, func(h registry.Host) bool { return h.Name == name }) {
			fc.HostsDropped = append(fc.HostsDropped, name)
		}
	}
	fc.hosts = !sameHosts(c.Hosts, after.Hosts)
	// Compare every variable the feature sets with the config's
	for _, e := range s.Env {
		old, ok := dc.Env(e.Name)
		// Name a variable the config lacks or sets otherwise
		switch {
		case !ok:
			fc.EnvAdded = append(fc.EnvAdded, e)
		case old != e.Value:
			fc.EnvChanged = append(fc.EnvChanged, EnvChange{Name: e.Name, Old: old, New: e.Value})
		}
	}
	return fc
}

// Apply writes the plan into the home and reports whether the allowlist changed.
func (p Plan) Apply(dir string) (bool, error) {
	// Hand back a stack that stays unwritten
	if err := home.Write(dir, p.Files); err != nil {
		return false, err
	}
	c, err := config.Load(dir)
	// Hand back settings that fail to load
	if err != nil {
		return false, err
	}
	dc, err := devcontainer.Load(dir)
	// Hand back a config that fails to load
	if err != nil {
		return false, err
	}
	hosts := false
	// Bring every feature to what its store ships
	for _, fc := range p.Features {
		dc.RetagFeature(fc.Package, fc.setup.Ref.String())
		// Set every variable the feature brings or changes
		for _, e := range fc.setup.Env {
			dc.SetEnv(e.Name, e.Value)
		}
		// Leave the allowlist alone where the feature's hosts stay
		if !fc.hosts {
			continue
		}
		hosts = true
		c.Release(fc.Package)
		// Open every host the feature now needs
		for _, h := range fc.setup.Hosts {
			// Name the host that breaks a rule, the files left as they were
			if err := c.Need(fc.Package, config.Change{Name: h.Name, Ports: h.Ports, Secret: h.Secret, Inject: h.Inject}); err != nil {
				return false, fmt.Errorf("%s needs %s: %s", fc.Package, h.Name, err.Error())
			}
		}
	}
	// Hand back a config that fails to save
	if err := dc.Save(dir); err != nil {
		return false, err
	}
	return hosts, c.Save(dir)
}

// neededBy lists the hosts that name a feature among their reasons.
func neededBy(c config.Config, pkg string) []string {
	var names []string
	// Look at every host
	for _, h := range c.Hosts {
		// Take a host the feature needs
		if slices.Contains(h.By, pkg) {
			names = append(names, h.Name)
		}
	}
	return names
}

// sameHosts reports whether two allowlists hold the same hosts, whatever order their reasons come in.
func sameHosts(a, b []config.Host) bool {
	key := func(hs []config.Host) []string {
		out := make([]string, len(hs))
		// Describe every host in one line, its reasons sorted
		for i, h := range hs {
			by := slices.Sorted(slices.Values(h.By))
			out[i] = fmt.Sprint(h.Name, h.Ports, h.Secret, h.Inject, by)
		}
		slices.Sort(out)
		return out
	}
	return slices.Equal(key(a), key(b))
}

// isMajor reports whether a tag names a major version alone.
func isMajor(tag string) bool {
	return tag != "" && strings.Trim(tag, "0123456789") == ""
}
