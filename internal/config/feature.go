package config

import (
	"fmt"
	"slices"
)

// Need opens a host for a feature and lists the feature among the host's reasons. A host the
// allowlist holds keeps its ports and gains the feature's, and a clash of secrets leaves it as it was.
func (c *Config) Need(by string, ch Change) error {
	// Turn away a host that breaks a rule of a single field
	if err := validate(ch); err != nil {
		return err
	}
	ch.Ports = normalPorts(ch.Ports)
	ch.Inject = normalInject(ch.Inject)
	want := newHost(ch)
	want.By = []string{by}
	// Turn away a host whose fields contradict each other
	if err := validateHost(want); err != nil {
		return err
	}
	i := slices.IndexFunc(c.Hosts, func(h Host) bool { return h.Name == ch.Name })
	// Add a host the allowlist lacks
	if i < 0 {
		c.Hosts = append(c.Hosts, want)
		c.order()
		return nil
	}
	h := c.Hosts[i]
	// Turn away a secret that would take the place of the one the host carries
	if h.Secret != "" && want.Secret != "" && h.Secret != want.Secret {
		return fmt.Errorf("%w secret %s on %s, which carries %s", ErrInvalid, want.Secret, h.Name, h.Secret)
	}
	h.Ports = normalPorts(append(slices.Clone(h.Ports), want.Ports...))
	// Take the feature's secret and where it goes
	if want.Secret != "" {
		h.Secret, h.Inject = want.Secret, normalInject(append(slices.Clone(h.Inject), want.Inject...))
	}
	// List the feature among the host's reasons once
	if !slices.Contains(h.By, by) {
		h.By = append(slices.Clone(h.By), by)
	}
	c.Hosts[i] = h
	return nil
}

// Release takes a feature off every host and hands back the hosts it drops once nothing needs them.
func (c *Config) Release(by string) []string {
	var dropped []string
	kept := c.Hosts[:0:0]
	// Take the feature off every host
	for _, h := range c.Hosts {
		h.By = slices.DeleteFunc(slices.Clone(h.By), func(b string) bool { return b == by })
		// Drop a host nothing needs any more
		if len(h.By) == 0 {
			dropped = append(dropped, h.Name)
			continue
		}
		kept = append(kept, h)
	}
	c.Hosts = kept
	return dropped
}
