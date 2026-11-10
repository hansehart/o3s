package config

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Diff is one field an update changed, with its value before and after.
type Diff struct {
	Field, Old, New string
}

// Allow adds a host to the allowlist, or updates the one it names. An update replaces each field
// the change gives and keeps the rest, and Allow reports every field it changed. The config stays
// as it was when the change breaks a rule.
func (c *Config) Allow(ch Change) ([]Diff, error) {
	// Turn away a change that breaks a rule of a single field
	if err := validate(ch); err != nil {
		return nil, err
	}
	ch.Ports = normalPorts(ch.Ports)
	ch.Inject = normalInject(ch.Inject)
	i := slices.IndexFunc(c.Hosts, func(h Host) bool { return h.Name == ch.Name })
	// Add a host the allowlist lacks, with its defaults
	if i < 0 {
		h := newHost(ch)
		// Turn away a host whose fields contradict each other
		if err := validateHost(h); err != nil {
			return nil, err
		}
		c.Hosts = append(c.Hosts, h)
		c.order()
		return nil, nil
	}
	h, diffs := update(c.Hosts[i], ch)
	// Turn away an update whose fields contradict each other
	if err := validateHost(h); err != nil {
		return nil, err
	}
	c.Hosts[i] = h
	return diffs, nil
}

// Remove takes the user off a host's needed-by list, and the host off the allowlist once nothing
// needs it. It hands back what still needs a host it keeps.
func (c *Config) Remove(name string) ([]string, error) {
	i := slices.IndexFunc(c.Hosts, func(h Host) bool { return h.Name == name })
	// Turn away a host the allowlist lacks
	if i < 0 {
		return nil, fmt.Errorf("%w %s", ErrUnknownHost, name)
	}
	h := c.Hosts[i]
	// Turn away a host the user never added, since its features own it
	if !slices.Contains(h.By, user) {
		return nil, fmt.Errorf("%s is %w: %s", name, ErrOwnedByFeatures, strings.Join(h.By, ", "))
	}
	h.By = slices.DeleteFunc(slices.Clone(h.By), func(b string) bool { return b == user })
	// Drop the host once nothing needs it
	if len(h.By) == 0 {
		c.Hosts = slices.Delete(c.Hosts, i, i+1)
		return nil, nil
	}
	c.Hosts[i] = h
	return h.By, nil
}

// newHost builds a host from a change, filling what the change leaves empty with its default.
func newHost(ch Change) Host {
	h := Host{Name: ch.Name, Ports: ch.Ports, Secret: ch.Secret, Inject: ch.Inject, By: []string{user}}
	// Allow the HTTPS port when no port is named
	if len(h.Ports) == 0 {
		h.Ports = []int{443}
	}
	// Inject a secret into the header when no site is named
	if h.Secret != "" && len(h.Inject) == 0 {
		h.Inject = []string{"header"}
	}
	return h
}

// update applies a change to a host, and reports each field it changed.
func update(h Host, ch Change) (Host, []Diff) {
	var diffs []Diff
	h.By = slices.Clone(h.By)
	// Replace the ports the change names
	if len(ch.Ports) > 0 && !slices.Equal(h.Ports, ch.Ports) {
		diffs = append(diffs, Diff{Field: "ports", Old: joinInts(h.Ports), New: joinInts(ch.Ports)})
		h.Ports = ch.Ports
	}
	// Replace the secret the change names
	if ch.Secret != "" && h.Secret != ch.Secret {
		diffs = append(diffs, Diff{Field: "secret", Old: h.Secret, New: ch.Secret})
		h.Secret = ch.Secret
	}
	inject := ch.Inject
	// Inject a newly given secret into the header when no site is named or kept
	if len(inject) == 0 && h.Secret != "" && len(h.Inject) == 0 {
		inject = []string{"header"}
	}
	// Replace the inject sites the change names
	if len(inject) > 0 && !slices.Equal(h.Inject, inject) {
		diffs = append(diffs, Diff{Field: "inject", Old: strings.Join(h.Inject, ","), New: strings.Join(inject, ",")})
		h.Inject = inject
	}
	// Make the host the user's as well
	if !slices.Contains(h.By, user) {
		old := strings.Join(h.By, ",")
		h.By = append(h.By, user)
		diffs = append(diffs, Diff{Field: "by", Old: old, New: strings.Join(h.By, ",")})
	}
	return h, diffs
}

// normalPorts sorts the ports and keeps each once.
func normalPorts(ports []int) []int {
	ports = slices.Clone(ports)
	slices.Sort(ports)
	return slices.Compact(ports)
}

// normalInject keeps each inject site once, in the order the proxy reads a request.
func normalInject(sites []string) []string {
	var out []string
	// Keep the sites in the proxy's own order
	for _, site := range injectSites {
		// Keep a site the change names
		if slices.Contains(sites, site) {
			out = append(out, site)
		}
	}
	return out
}

// joinInts writes numbers as one comma-separated value.
func joinInts(ns []int) string {
	parts := make([]string, len(ns))
	// Write every number
	for i, n := range ns {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ",")
}
