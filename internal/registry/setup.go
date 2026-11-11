package registry

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// FeaturesRoot is where the cage keeps every feature's state dir.
const FeaturesRoot = "/home/ubuntu/features/"

// ErrUnknownFeature marks a feature its store or the home lacks.
var ErrUnknownFeature = errors.New("unknown feature")

// Setup is what a feature sets up in a home: its tag, its options, the hosts it needs and its env.
type Setup struct {
	Ref     Ref
	Version string
	Options map[string]string
	Hosts   []Host
	Env     []Env
}

// Resolver works out what features set up and reads each store once.
type Resolver struct {
	stores map[string][]Feature
}

// Resolve finds a feature in its store and works out what it sets up, its tag the store's major
// version unless the id names one.
func (r *Resolver) Resolve(ctx context.Context, ref Ref) (Setup, error) {
	features, err := r.collection(ctx, ref.Store)
	// Name the store whose features stay out of reach
	if err != nil {
		return Setup{}, fmt.Errorf("read the features of %s: %w", ref.Store, err)
	}
	i := slices.IndexFunc(features, func(f Feature) bool { return f.ID == ref.ID })
	// Turn away a feature the store lacks
	if i < 0 {
		return Setup{}, fmt.Errorf("%w %s", ErrUnknownFeature, ref.Package())
	}
	f := features[i]
	// Follow the store's major version where the id names no tag
	if ref.Tag == "" {
		ref.Tag = Major(f.Version)
	}
	dir := FeaturesRoot + ref.Package()
	s := Setup{Ref: ref, Version: f.Version, Hosts: f.Hosts}
	// Give the feature its state dir where it takes one
	if slices.Contains(f.Options, "stateDir") {
		s.Options = map[string]string{"stateDir": dir}
	}
	// Fill the state dir into every variable
	for _, e := range f.Env {
		s.Env = append(s.Env, Env{Name: e.Name, Value: strings.ReplaceAll(e.Value, "${stateDir}", dir)})
	}
	return s, nil
}

// collection hands back a store's features, read from the registry the first time alone.
func (r *Resolver) collection(ctx context.Context, store string) ([]Feature, error) {
	// Take the features read before
	if features, ok := r.stores[store]; ok {
		return features, nil
	}
	features, err := Collection(ctx, store)
	// Hand back a store that fails to read, to be asked again next time
	if err != nil {
		return nil, err
	}
	// Start the memory on the first read
	if r.stores == nil {
		r.stores = map[string][]Feature{}
	}
	r.stores[store] = features
	return features, nil
}

// Major reads the major part of a version, and latest for a feature without one.
func Major(version string) string {
	m, _, _ := strings.Cut(version, ".")
	// Follow the newest release of a feature without a version
	if m == "" {
		return "latest"
	}
	return m
}

// PortsOf hands back the ports a feature names for a host, the HTTPS port where it names none.
func PortsOf(h Host) []int {
	// Take the HTTPS port where the feature names none
	if len(h.Ports) == 0 {
		return []int{443}
	}
	return h.Ports
}
