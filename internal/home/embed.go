package home

import "embed"

// source is where the binary keeps the stack it writes into every home's .devcontainer.
const source = "stack"

// files holds the stack and the templates o3s seeds a home's own files from.
//
//go:embed stack templates
var files embed.FS
