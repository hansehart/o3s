// Package o3s holds the stack an o3s home runs, so the o3s binary carries it into every home it
// makes.
package o3s

import "embed"

// Stack is the dev container setup every home runs: its config, compose file, confinement profiles,
// cage scripts, lifecycle hooks, proxy addon and seeds. The overlay that builds images from this
// checkout stays behind, since a home runs the published images.
//
//go:embed .devcontainer/apparmor.conf .devcontainer/compose.yaml .devcontainer/devcontainer.json .devcontainer/seccomp.json
//go:embed .devcontainer/cage .devcontainer/lifecycle .devcontainer/proxy .devcontainer/templates
var Stack embed.FS
