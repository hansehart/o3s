package cli

import (
	"runtime/debug"
	"strings"
)

// Version is the version a release build sets, and empty in any other build.
var Version string

// Digests are the digests of the images a release build was built with, by image, and empty in any other build.
var Digests map[string]string

// version names the running build.
func version() string {
	info, _ := debug.ReadBuildInfo()
	return versionOf(Version, info)
}

// versionOf names a build by the version its release set, else by the module version it was installed at, else as dev.
func versionOf(released string, info *debug.BuildInfo) string {
	// Take the version the release set
	if released != "" {
		return released
	}
	// Take the module version a go install at a tag records
	if info != nil && strings.HasPrefix(info.Main.Version, "v") {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "dev"
}
