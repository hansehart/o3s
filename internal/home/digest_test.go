package home_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/home"
)

// Digests of a release's images, as a release binary carries them.
var (
	cageDigest    = "sha256:" + strings.Repeat("a", 64)
	gatewayDigest = "sha256:" + strings.Repeat("b", 64)
)

// aRelease sets the version and digests a release binary carries, until the test ends.
func aRelease(t *testing.T, version string, digests map[string]string) {
	t.Helper()
	home.Version, home.Digests = version, digests
	t.Cleanup(func() { home.Version, home.Digests = "dev", nil })
}

// composeOf reads the compose file of a home.
func composeOf(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ".devcontainer", "compose.yaml"))
	// Stop when the compose file fails to read
	if err != nil {
		t.Fatalf("ReadFile() = %v", err)
	}
	return string(data)
}

// A release pins each image it carries a well-formed digest for, and tags the rest with its version alone.
func TestInitPinsTheImagesByDigest(t *testing.T) {
	tests := []struct {
		name    string
		digests map[string]string
		want    []string
	}{
		{
			"both digests",
			map[string]string{"ghcr.io/hansehart/o3s-cage": cageDigest, "ghcr.io/hansehart/o3s-gateway": gatewayDigest},
			[]string{"image: ghcr.io/hansehart/o3s-cage:0.7.0@" + cageDigest + "\n", "image: ghcr.io/hansehart/o3s-gateway:0.7.0@" + gatewayDigest + "\n"},
		},
		{
			"no digests",
			nil,
			[]string{"image: ghcr.io/hansehart/o3s-cage:0.7.0\n", "image: ghcr.io/hansehart/o3s-gateway:0.7.0\n"},
		},
		{
			"a malformed digest",
			map[string]string{"ghcr.io/hansehart/o3s-cage": "sha256:short", "ghcr.io/hansehart/o3s-gateway": gatewayDigest},
			[]string{"image: ghcr.io/hansehart/o3s-cage:0.7.0\n", "image: ghcr.io/hansehart/o3s-gateway:0.7.0@" + gatewayDigest + "\n"},
		},
	}
	// Write a home for every release
	for _, tt := range tests {
		aRelease(t, "0.7.0", tt.digests)

		compose := composeOf(t, anInitializedHome(t))

		// Require every image line
		for _, line := range tt.want {
			// Report a line the compose file lacks
			if !strings.Contains(compose, line) {
				t.Errorf("%s: compose lacks %q", tt.name, line)
			}
		}
	}
}

// An update names the tags that move, the digests left out of the display.
func TestImagesShowsTagsWithoutDigests(t *testing.T) {
	aRelease(t, "0.7.0", map[string]string{"ghcr.io/hansehart/o3s-cage": cageDigest, "ghcr.io/hansehart/o3s-gateway": gatewayDigest})
	dir := anInitializedHome(t)
	aRelease(t, "0.8.0", map[string]string{"ghcr.io/hansehart/o3s-cage": gatewayDigest, "ghcr.io/hansehart/o3s-gateway": cageDigest})

	moved, err := home.Images(dir)

	// Stop when the images fail to read
	if err != nil {
		t.Fatalf("Images() = %v", err)
	}
	want := []home.ImageChange{{Name: "o3s-cage", Old: "0.7.0", New: "0.8.0"}, {Name: "o3s-gateway", Old: "0.7.0", New: "0.8.0"}}
	// Require the tags alone
	if diff := cmp.Diff(want, moved); diff != "" {
		t.Errorf("Images() (-want +got):\n%s", diff)
	}
}

// A home on the same release without digests names no image change, and the refresh names the compose file that brings the pin.
func TestRefreshBringsTheDigestsToATaggedHome(t *testing.T) {
	aRelease(t, "0.7.0", nil)
	dir := anInitializedHome(t)
	aRelease(t, "0.7.0", map[string]string{"ghcr.io/hansehart/o3s-cage": cageDigest, "ghcr.io/hansehart/o3s-gateway": gatewayDigest})

	moved, err := home.Images(dir)
	// Stop when the images fail to read
	if err != nil {
		t.Fatalf("Images() = %v", err)
	}
	changes, err := home.Refresh(dir)

	// Stop when the refresh fails
	if err != nil {
		t.Fatalf("Refresh() = %v", err)
	}
	// Require no image change on the same release
	if len(moved) != 0 {
		t.Errorf("Images() = %v, want none", moved)
	}
	// Require the compose file named for the refresh that brings the pin
	if !slices.Contains(paths(changes), "~ compose.yaml") {
		t.Errorf("Refresh() = %v, want ~ compose.yaml", paths(changes))
	}
}
