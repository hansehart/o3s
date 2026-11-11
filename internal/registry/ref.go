package registry

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// ErrInvalid marks a feature id that names no feature in a store.
var ErrInvalid = errors.New("invalid feature")

// Ref names a feature as a store, the feature's id in it, and an optional tag.
type Ref struct {
	Store string
	ID    string
	Tag   string
}

// ParseRef reads a full feature id with an optional tag.
func ParseRef(s string) (Ref, error) {
	bad := fmt.Errorf("%w %q: name it as a store and a feature, as in ghcr.io/devcontainers/features/go:1", ErrInvalid, s)
	store, id, ok := strings.CutLast(s, "/")
	// Turn away an id that names no store
	if !ok {
		return Ref{}, bad
	}
	r := Ref{Store: store, ID: id}
	// Split the tag off the feature's id
	if id, tag, ok := strings.Cut(r.ID, ":"); ok {
		r.ID, r.Tag = id, tag
		// Turn away a tag that breaks the registry's rules
		if _, err := name.NewTag(r.Package()+":"+tag, name.StrictValidation); err != nil {
			return Ref{}, bad
		}
	}
	// Turn away a store that names no repository in full
	if _, err := name.NewRepository(r.Store, name.StrictValidation); err != nil {
		return Ref{}, bad
	}
	// Turn away a feature id that breaks the registry's rules
	if _, err := name.NewRepository(r.Package(), name.StrictValidation); err != nil {
		return Ref{}, bad
	}
	return r, nil
}

// Package is the feature's id without its tag, the name a host lists it by.
func (r Ref) Package() string {
	return r.Store + "/" + r.ID
}

// String writes the full id with its tag when it carries one.
func (r Ref) String() string {
	// Name the package alone when no tag is set
	if r.Tag == "" {
		return r.Package()
	}
	return r.Package() + ":" + r.Tag
}
