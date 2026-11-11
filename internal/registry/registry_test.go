package registry_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"github.com/hansehart/o3s/internal/registry"
	"github.com/hansehart/o3s/internal/registry/registrytest"
)

// aRegistry hands a test an in-memory registry that lives as long as the test.
func aRegistry(t *testing.T) *registrytest.Registry {
	t.Helper()
	r, err := registrytest.New()
	// Stop when the registry fails to come up
	if err != nil {
		t.Fatalf("Setup: registrytest.New() = %v", err)
	}
	t.Cleanup(r.Close)
	return r
}

// A store's collection yields the ids of its features in order.
func TestCollectionReadsTheFeatureIdsInOrder(t *testing.T) {
	r := aRegistry(t)
	// Stop when the store fails to publish
	if err := r.Push("features", "beta", "alpha"); err != nil {
		t.Fatalf("Setup: Push(features) = %v", err)
	}

	got, err := registry.Collection(t.Context(), r.Host+"/features")

	// Stop when the collection fails to read
	if err != nil {
		t.Fatalf("Collection(features) = %v", err)
	}
	// Report ids that differ from the published ones
	if diff := cmp.Diff([]string{"alpha", "beta"}, got); diff != "" {
		t.Errorf("Collection(features) mismatch (-want +got):\n%s", diff)
	}
}

// A store the registry lacks is turned away.
func TestCollectionTurnsAwayAnAbsentStore(t *testing.T) {
	r := aRegistry(t)

	_, err := registry.Collection(t.Context(), r.Host+"/absent")

	// Require an error for a store that holds nothing
	if err == nil {
		t.Errorf("Collection(absent) = nil, want an error")
	}
}

// An artifact without a collection layer is named as no store.
func TestCollectionTurnsAwayAnArtifactWithoutACollection(t *testing.T) {
	r := aRegistry(t)
	// Stop when the artifact fails to publish
	if err := r.PushLayer("image", static.NewLayer([]byte("{}"), types.OCILayer)); err != nil {
		t.Fatalf("Setup: PushLayer(image) = %v", err)
	}

	_, err := registry.Collection(t.Context(), r.Host+"/image")

	// Require the error that names a missing collection
	if !errors.Is(err, registry.ErrNoCollection) {
		t.Errorf("Collection(image) = %v, want ErrNoCollection", err)
	}
}
