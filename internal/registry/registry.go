// Package registry reads the feature stores o3s offers features from.
package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

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

// Collection fetches a store's collection with the user's registry logins and hands back its feature ids in order.
func Collection(ctx context.Context, store string) ([]string, error) {
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
		return featureIDs(l)
	}
	return nil, fmt.Errorf("%s %w", store, ErrNoCollection)
}

// featureIDs reads the ids of the features a collection layer lists, in order.
func featureIDs(l v1.Layer) ([]string, error) {
	rc, err := l.Compressed()
	// Hand back a layer that fails to open
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var c struct {
		Features []struct {
			ID string `json:"id"`
		} `json:"features"`
	}
	// Hand back a collection that fails to decode
	if err := json.NewDecoder(rc).Decode(&c); err != nil {
		return nil, fmt.Errorf("read the collection: %w", err)
	}
	ids := make([]string, len(c.Features))
	// Take every feature's id
	for i, f := range c.Features {
		ids[i] = f.ID
	}
	slices.Sort(ids)
	return ids, nil
}
