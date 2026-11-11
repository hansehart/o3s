package registry_test

import (
	"errors"
	"testing"

	"github.com/hansehart/o3s/internal/registry"
)

// A feature id splits into its store, its package and the tag it names.
func TestParseRefSplitsAFeatureID(t *testing.T) {
	tests := []struct {
		in   string
		want registry.Ref
	}{
		{"registry.example.com/features/tool", registry.Ref{Store: "registry.example.com/features", ID: "tool"}},
		{"registry.example.com/features/tool:1", registry.Ref{Store: "registry.example.com/features", ID: "tool", Tag: "1"}},
		{"127.0.0.1:5000/team/features/tool:1.2", registry.Ref{Store: "127.0.0.1:5000/team/features", ID: "tool", Tag: "1.2"}},
	}
	// Check every id
	for _, tt := range tests {
		got, err := registry.ParseRef(tt.in)

		// Report an id that splits otherwise
		if err != nil || got != tt.want {
			t.Errorf("ParseRef(%q) = %+v, %v, want %+v", tt.in, got, err, tt.want)
		}
	}
}

// A feature's package is its id without the tag.
func TestRefPackageLeavesTheTagOut(t *testing.T) {
	r := registry.Ref{Store: "registry.example.com/features", ID: "tool", Tag: "1"}

	// Report a package that differs
	if got, want := r.Package(), "registry.example.com/features/tool"; got != want {
		t.Errorf("Package() = %q, want %q", got, want)
	}
}

// An id that names no feature in a store is turned away.
func TestParseRefTurnsAwayAMalformedID(t *testing.T) {
	// Check every malformed id
	for _, in := range []string{"", "tool", "registry.example.com/tool", "registry.example.com/features/Tool", "registry.example.com/features/tool:", "registry.example.com/features/tool@sha256:0"} {
		_, err := registry.ParseRef(in)

		// Require the error that marks a malformed id
		if !errors.Is(err, registry.ErrInvalid) {
			t.Errorf("ParseRef(%q) = %v, want ErrInvalid", in, err)
		}
	}
}
