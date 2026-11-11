package config

import (
	"fmt"
	"net/netip"
	"regexp"
	"slices"
	"strings"
)

var (
	// domainName matches a lowercase domain, optionally led by a wildcard label.
	domainName = regexp.MustCompile(`^(\*\.)?[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)
	// envName matches an uppercase environment variable name.
	envName = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
	// injectSites lists where the proxy may place a secret in a request.
	injectSites = []string{"header", "query", "body"}
)

// validate holds a change to the allowlist's rules, naming the first rule it breaks.
func validate(ch Change) error {
	// Require a host in one of the forms the gateway resolves
	if !validName(ch.Name) {
		return fmt.Errorf("%w host %q, want example.com, *.example.com, an IPv4 address or an IPv4 CIDR", ErrInvalid, ch.Name)
	}
	// Require every port inside the TCP range
	for _, p := range ch.Ports {
		// Name the first port outside the range
		if p < 1 || p > 65535 {
			return fmt.Errorf("%w port %d, want 1-65535", ErrInvalid, p)
		}
	}
	// Require a secret to read as an environment variable
	if ch.Secret != "" && !envName.MatchString(ch.Secret) {
		return fmt.Errorf("%w secret %q, want an uppercase env name", ErrInvalid, ch.Secret)
	}
	// Require every inject site to be one the proxy knows
	for _, site := range ch.Inject {
		// Name the first site the proxy lacks
		if !slices.Contains(injectSites, site) {
			return fmt.Errorf("%w inject %q, want header, query or body", ErrInvalid, site)
		}
	}
	return nil
}

// validateHost holds a host, as a change leaves it, to the rules that span its fields.
func validateHost(h Host) error {
	// Require a secret wherever inject sites are named
	if len(h.Inject) > 0 && h.Secret == "" {
		return fmt.Errorf("%w inject without a secret on %s", ErrInvalid, h.Name)
	}
	return nil
}

// IsSecretName reports whether a name reads as the environment variable a secret travels in.
func IsSecretName(name string) bool {
	return envName.MatchString(name)
}

// validName reports whether a host is a domain, a wildcard domain, an IPv4 address or an IPv4 CIDR.
func validName(name string) bool {
	// Accept an IPv4 range
	if prefix, err := netip.ParsePrefix(name); err == nil {
		return prefix.Addr().Is4()
	}
	// Accept an IPv4 address
	if addr, err := netip.ParseAddr(name); err == nil {
		return addr.Is4()
	}
	return !strings.Contains(name, ":") && domainName.MatchString(name)
}
