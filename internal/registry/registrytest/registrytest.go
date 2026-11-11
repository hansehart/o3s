// Package registrytest raises an in-memory registry that holds feature stores, the way httptest
// raises a server for HTTP tests.
package registrytest

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"net/url"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// CollectionMediaType is the media type the Dev Container spec gives a store's collection.
const CollectionMediaType types.MediaType = "application/vnd.devcontainers.collection.layer.v1+json"

// Registry is an in-memory registry and the host a store id names it by.
type Registry struct {
	Host   string
	server *httptest.Server
}

// New raises an empty registry until Close.
func New() (*Registry, error) {
	s := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	u, err := url.Parse(s.URL)
	// Hand back a server address that fails to parse
	if err != nil {
		s.Close()
		return nil, err
	}
	return &Registry{Host: u.Host, server: s}, nil
}

// Close stops the registry.
func (r *Registry) Close() { r.server.Close() }

// Push publishes a store that holds a collection of the features named.
func (r *Registry) Push(repo string, ids ...string) error {
	features := make([]string, len(ids))
	// Describe every feature by its id alone
	for i, id := range ids {
		features[i] = fmt.Sprintf(`{"id": %q}`, id)
	}
	return r.PushFeatures(repo, features...)
}

// PushFeatures publishes a store whose collection holds each feature's metadata as given.
func (r *Registry) PushFeatures(repo string, features ...string) error {
	raw := make([]json.RawMessage, len(features))
	// Take every feature's metadata as it stands
	for i, f := range features {
		raw[i] = json.RawMessage(f)
	}
	body, err := json.Marshal(map[string]any{"sourceInformation": map[string]string{}, "features": raw})
	// Hand back a collection that fails to encode
	if err != nil {
		return err
	}
	return r.PushLayer(repo, static.NewLayer(body, CollectionMediaType))
}

// PushLayer publishes a store as an artifact that holds one layer.
func (r *Registry) PushLayer(repo string, layer v1.Layer) error {
	img, err := mutate.AppendLayers(mutate.MediaType(empty.Image, types.OCIManifestSchema1), layer)
	// Hand back an artifact that fails to build
	if err != nil {
		return err
	}
	ref, err := name.ParseReference(fmt.Sprintf("%s/%s:latest", r.Host, repo))
	// Hand back a store name that fails to parse
	if err != nil {
		return err
	}
	return remote.Write(ref, img)
}
