package host

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hansehart/o3s/internal/atomicfile"
	"github.com/hansehart/o3s/internal/paths"
	"github.com/hansehart/o3s/internal/secrets"
)

// Env writes the values compose reads at create time, keeping the marker the cage already holds.
func Env(home string) error {
	path := filepath.Join(home, ".devcontainer", ".env")
	marker := markerIn(path)
	// Draw a new marker where the home holds none
	if marker == "" {
		b := make([]byte, 32)
		// Hand back randomness that fails to draw
		if _, err := rand.Read(b); err != nil {
			return err
		}
		marker = "o3s-" + hex.EncodeToString(b)
	}
	secretsFile, err := secrets.Path()
	// Hand back a secrets folder whose place stays unknown
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	fmt.Fprintln(&buf, "# Generated on every start. Put your own variables in devcontainer.json.")
	fmt.Fprintln(&buf)
	fmt.Fprintf(&buf, "O3S_MARKER=%s\nO3S_UID=%d\nO3S_GID=%d\n", marker, os.Getuid(), os.Getgid())
	fmt.Fprintf(&buf, "O3S_PROXY_DIR=%s\nO3S_SECRETS_DIR=%s\nO3S_SOCKET_DIR=%s\n", paths.Proxy(), filepath.Dir(secretsFile), filepath.Join(paths.Home(home), "docker-host"))
	return atomicfile.Write(path, buf.Bytes(), 0o644)
}

// markerIn reads the marker an env file holds, and empty where it holds none.
func markerIn(path string) string {
	data, err := os.ReadFile(path)
	// Read a missing file as one without a marker
	if err != nil {
		return ""
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	// Look at every line for the marker
	for sc.Scan() {
		// Take the marker's value
		if value, ok := strings.CutPrefix(sc.Text(), "O3S_MARKER="); ok {
			return value
		}
	}
	return ""
}
