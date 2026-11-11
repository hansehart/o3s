// Package registry reads the feature stores o3s offers features from.
package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// collection is the media type the Dev Container spec gives a store's collection.
const collection types.MediaType = "application/vnd.devcontainers.collection.layer.v1+json"

// ErrNoCollection marks an artifact that holds no feature collection.
var ErrNoCollection = errors.New("holds no feature collection")

// Feature is what a store tells about one of its features.
type Feature struct {
	ID      string
	Version string
	Options []string
	Hosts   []Host
	Env     []Env
}

// Host is a host a feature needs, with the secret the proxy injects there.
type Host struct {
	Name   string   `json:"name"`
	Ports  []int    `json:"ports"`
	Secret string   `json:"secret"`
	Inject []string `json:"inject"`
}

// Env is a variable a feature sets for every process in the cage.
type Env struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Collection fetches a store's collection with the user's registry logins and hands back its features in id order.
func Collection(ctx context.Context, store string) ([]Feature, error) {
	ref, err := name.ParseReference(store + ":latest")
	// Hand back a store name that fails to parse
	if err != nil {
		return nil, err
	}
	img, err := remote.Image(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(authn.DefaultKeychain))
	// Hand back a store the registry fails to serve
	if err != nil {
		return nil, err
	}
	layers, err := img.Layers()
	// Hand back an artifact whose layers fail to read
	if err != nil {
		return nil, err
	}
	// Read the first layer that holds the collection
	for _, l := range layers {
		mt, err := l.MediaType()
		// Hand back a layer whose type fails to read
		if err != nil {
			return nil, err
		}
		// Look further while the layer holds something else
		if mt != collection {
			continue
		}
		return features(l)
	}
	return nil, fmt.Errorf("%s %w", store, ErrNoCollection)
}

// features reads the features a collection layer lists, in id order.
func features(l v1.Layer) ([]Feature, error) {
	rc, err := l.Compressed()
	// Hand back a layer that fails to open
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var c struct {
		Features []struct {
			ID             string                     `json:"id"`
			Version        string                     `json:"version"`
			Options        map[string]json.RawMessage `json:"options"`
			Customizations struct {
				O3S struct {
					Hosts []Host `json:"hosts"`
					Env   []Env  `json:"env"`
				} `json:"o3s"`
			} `json:"customizations"`
		} `json:"features"`
	}
	// Hand back a collection that fails to decode
	if err := json.NewDecoder(rc).Decode(&c); err != nil {
		return nil, fmt.Errorf("read the collection: %w", err)
	}
	fs := make([]Feature, len(c.Features))
	// Take what every feature tells
	for i, f := range c.Features {
		o := f.Customizations.O3S
		fs[i] = Feature{ID: f.ID, Version: f.Version, Hosts: o.Hosts, Env: o.Env}
		// List the options the feature offers, in order
		for name := range f.Options {
			fs[i].Options = append(fs[i].Options, name)
		}
		slices.Sort(fs[i].Options)
	}
	slices.SortFunc(fs, func(a, b Feature) int { return strings.Compare(a.ID, b.ID) })
	return fs, nil
}
