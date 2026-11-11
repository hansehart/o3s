package stacktest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Parsed is what the proxy's addon reads from a policy and its secrets.
type Parsed struct {
	Keys     []string            `json:"keys"`
	Allowed  []string            `json:"allowed"`
	Injected map[string]*string  `json:"injected"`
	Sites    map[string][]string `json:"sites"`
}

// readState prints the addon's view of its inputs as one line of JSON.
const readState = `import json, sys
sys.path.insert(0, "/addons")
import inject
a = inject.InjectCredentials()
print(json.dumps({
    "keys": sorted(a._keys),
    "allowed": sorted(a._allowed),
    "injected": dict(sorted(a._hosts.items())),
    "sites": {k: sorted(v) for k, v in sorted(a._sites.items())},
}))`

// ProxyReads runs the proxy's addon from this branch in its own image and returns what it read.
func ProxyReads(tb testing.TB, policy, secrets string) (Parsed, error) {
	tb.Helper()
	dir := tb.TempDir()
	// Stop when the inputs stay unwritten
	for name, content := range map[string]string{"o3s.json": policy, "secrets.env": secrets} {
		// Stop at the first input that fails to write
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			tb.Fatalf("Setup: write %s: %v", name, err)
		}
	}
	addon := filepath.Join(Repo(tb), ".devcontainer/proxy/inject.py")
	out, err := run(tb.Context(), tb, "docker", "run", "--rm",
		"--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		"--entrypoint", "python3",
		"-e", "O3S_MARKER=o3s-stacktest",
		"-e", "O3S_CONFIG_FILE=/inputs/o3s.json",
		"-e", "O3S_SECRETS_FILE=/inputs/secrets.env",
		"-v", dir+":/inputs:ro",
		"-v", addon+":/addons/inject.py:ro",
		proxyImage(tb), "-c", readState)
	// Hand back an addon that fails on its inputs
	if err != nil {
		return Parsed{}, err
	}
	var parsed Parsed
	// Hand back output in another shape than the addon's view
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return Parsed{}, fmt.Errorf("read the addon's view %q: %w", out, err)
	}
	return parsed, nil
}

// proxyImage reads the image the stack runs its proxy from.
func proxyImage(tb testing.TB) string {
	tb.Helper()
	anEnv(tb)
	out, err := compose(tb.Context(), tb, "config", "--format", "json")
	// Stop when the stack's model stays unread
	if err != nil {
		tb.Fatalf("Setup: %v", err)
	}
	var model struct {
		Services map[string]struct {
			Image string `json:"image"`
		} `json:"services"`
	}
	// Stop when the stack's model reads in another shape
	if err := json.Unmarshal([]byte(out), &model); err != nil {
		tb.Fatalf("Setup: read the stack's model: %v", err)
	}
	return model.Services["proxy"].Image
}
