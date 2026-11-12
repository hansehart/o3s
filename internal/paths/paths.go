// Package paths names the folders o3s keeps on a host, following the XDG Base Directory Specification.
package paths

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

// Config is where the user's own o3s files live, such as the secrets.
func Config() string {
	return base("XDG_CONFIG_HOME", ".config")
}

// State is where o3s keeps what it generates on this host and can make again.
func State() string {
	return base("XDG_STATE_HOME", ".local/state")
}

// Data is where o3s keeps the tools it installs and manages.
func Data() string {
	return base("XDG_DATA_HOME", ".local/share")
}

// Proxy is the folder of the proxy's CA, the only one the proxy mounts.
func Proxy() string {
	return filepath.Join(State(), "proxy")
}

// Devcontainers is where o3s keeps its own devcontainer CLI.
func Devcontainers() string {
	return filepath.Join(Data(), "devcontainers")
}

// Home is the folder o3s keeps for one home, named after a short hash of the home's path.
func Home(dir string) string {
	sum := sha256.Sum256([]byte(dir))
	return filepath.Join(State(), hex.EncodeToString(sum[:])[:12])
}

// base hands back a base folder from its XDG variable where that is absolute, else from its default below the home folder.
func base(variable, fallback string) string {
	// Follow the variable where it names an absolute folder, as the spec asks
	if dir := os.Getenv(variable); filepath.IsAbs(dir) {
		return filepath.Join(dir, "o3s")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, fallback, "o3s")
}
