package config_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/config"
)

// A new store joins the list once, and a repeat reports it as known.
func TestAddStoreAddsAStoreOnce(t *testing.T) {
	var c config.Config

	added, err := c.AddStore("registry.example.com/features")
	// Stop when the store is turned away
	if err != nil || !added {
		t.Fatalf("AddStore(registry.example.com/features) = %v, %v, want true, nil", added, err)
	}
	again, err := c.AddStore("registry.example.com/features")

	// Require a repeat to report the store as known
	if err != nil || again {
		t.Errorf("AddStore again = %v, %v, want false, nil", again, err)
	}
	// Require the store listed once
	if diff := cmp.Diff([]config.Store{{ID: "registry.example.com/features"}}, c.Stores); diff != "" {
		t.Errorf("stores (-want +got):\n%s", diff)
	}
}

// A store id that names no repository is turned away and leaves the list as it was.
func TestAddStoreTurnsAwayAMalformedID(t *testing.T) {
	// Check every malformed id
	for _, id := range []string{"", "Registry.example.com/Features", "registry.example.com/features:latest", "registry.example.com/features@sha256:0"} {
		var c config.Config

		_, err := c.AddStore(id)

		// Require the error that marks a broken rule
		if !errors.Is(err, config.ErrInvalid) {
			t.Errorf("AddStore(%q) = %v, want ErrInvalid", id, err)
		}
		// Require the list to stay empty
		if len(c.Stores) != 0 {
			t.Errorf("AddStore(%q) stores = %v, want none", id, c.Stores)
		}
	}
}

// Removing a store drops it from the list.
func TestRemoveStoreDropsTheStore(t *testing.T) {
	c := config.Config{Stores: []config.Store{{ID: "registry.example.com/features"}, {ID: "registry.example.org/features"}}}

	err := c.RemoveStore("registry.example.com/features")

	// Stop when the store is turned away
	if err != nil {
		t.Fatalf("RemoveStore(registry.example.com/features) = %v", err)
	}
	// Require the other store alone to stay
	if diff := cmp.Diff([]config.Store{{ID: "registry.example.org/features"}}, c.Stores); diff != "" {
		t.Errorf("stores (-want +got):\n%s", diff)
	}
}

// Removing a store the list lacks is turned away.
func TestRemoveStoreTurnsAwayAnUnknownStore(t *testing.T) {
	var c config.Config

	err := c.RemoveStore("registry.example.com/features")

	// Require the error that names an unknown store
	if !errors.Is(err, config.ErrUnknownStore) {
		t.Errorf("RemoveStore(registry.example.com/features) = %v, want ErrUnknownStore", err)
	}
}
