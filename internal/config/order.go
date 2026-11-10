package config

import (
	"cmp"
	"net/netip"
	"slices"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// order puts the hosts in apex order and the stores in id order, so a diff of the file shows each
// change alone.
func (c *Config) order() {
	slices.SortStableFunc(c.Hosts, func(a, b Host) int { return compareHosts(a.Name, b.Name) })
	slices.SortStableFunc(c.Stores, func(a, b Store) int { return strings.Compare(a.ID, b.ID) })
}

// hostKey is where a host sorts: domains before addresses, then by apex, the apex itself first.
type hostKey struct {
	address bool
	apex    string
	sub     bool
	name    string
	prefix  netip.Prefix
}

// keyOf places a host in the apex order.
func keyOf(name string) hostKey {
	// Place a range by its first address and its width
	if prefix, err := netip.ParsePrefix(name); err == nil {
		return hostKey{address: true, prefix: prefix}
	}
	// Place a single address as a range of one
	if addr, err := netip.ParseAddr(name); err == nil {
		return hostKey{address: true, prefix: netip.PrefixFrom(addr, addr.BitLen())}
	}
	bare := strings.TrimPrefix(name, "*.")
	apex, err := publicsuffix.EffectiveTLDPlusOne(bare)
	// Treat a name without a registrable domain as its own apex
	if err != nil {
		apex = bare
	}
	return hostKey{apex: apex, sub: name != apex, name: bare}
}

// compareHosts orders two hosts by apex, the apex before its subdomains, and addresses last.
func compareHosts(a, b string) int {
	ka, kb := keyOf(a), keyOf(b)
	// Put addresses after every domain
	if ka.address != kb.address {
		return boolOrder(ka.address, kb.address)
	}
	// Order addresses by their start, then by their width
	if ka.address {
		return cmp.Or(ka.prefix.Addr().Compare(kb.prefix.Addr()), cmp.Compare(ka.prefix.Bits(), kb.prefix.Bits()))
	}
	return cmp.Or(strings.Compare(ka.apex, kb.apex), boolOrder(ka.sub, kb.sub), strings.Compare(ka.name, kb.name))
}

// boolOrder puts false before true.
func boolOrder(a, b bool) int {
	// Keep equal flags in place
	if a == b {
		return 0
	}
	// Put the false one first
	if !a {
		return -1
	}
	return 1
}
