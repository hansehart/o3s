// Package o3s holds the stack the o3s binary ships to every home.
package o3s

import "embed"

// Stack is the dev container setup every home runs.
//
//go:embed .devcontainer/apparmor.conf .devcontainer/compose.yaml .devcontainer/devcontainer.json .devcontainer/seccomp.json
//go:embed .devcontainer/cage .devcontainer/lifecycle .devcontainer/proxy .devcontainer/templates
var Stack embed.FS
