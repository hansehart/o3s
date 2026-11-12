package cli

import (
	"runtime/debug"
	"testing"
)

// The version the release sets wins, a module build reads its own, and a checkout shows dev.
func TestVersionNamesTheBuild(t *testing.T) {
	module := &debug.BuildInfo{Main: debug.Module{Version: "v0.7.0"}}
	checkout := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}
	tests := []struct {
		name     string
		released string
		info     *debug.BuildInfo
		want     string
	}{
		{"a release", "0.7.0", checkout, "0.7.0"},
		{"go install at a tag", "", module, "0.7.0"},
		{"a checkout", "", checkout, "dev"},
		{"a build without its info", "", nil, "dev"},
	}
	// Check every kind of build
	for _, tt := range tests {
		// Report a version that differs
		if got := versionOf(tt.released, tt.info); got != tt.want {
			t.Errorf("%s: versionOf() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
