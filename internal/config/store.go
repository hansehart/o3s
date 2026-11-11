package config

import (
	"fmt"
	"slices"

	"github.com/google/go-containerregistry/pkg/name"
)

// AddStore adds a store to the list and reports whether the list lacked it.
func (c *Config) AddStore(id string) (bool, error) {
	// Turn away an id that names no repository in full
	if _, err := name.NewRepository(id, name.StrictValidation); err != nil {
		return false, fmt.Errorf("%w store %q: name it as a registry and a repository, as in ghcr.io/devcontainers/features", ErrInvalid, id)
	}
	// Keep a store the list already holds
	if slices.Contains(c.Stores, Store{ID: id}) {
		return false, nil
	}
	c.Stores = append(c.Stores, Store{ID: id})
	return true, nil
}

// RemoveStore drops a store from the list.
func (c *Config) RemoveStore(id string) error {
	i := slices.Index(c.Stores, Store{ID: id})
	// Turn away a store the list lacks
	if i < 0 {
		return fmt.Errorf("%w %s", ErrUnknownStore, id)
	}
	c.Stores = slices.Delete(c.Stores, i, i+1)
	return nil
}
