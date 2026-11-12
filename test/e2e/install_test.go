//go:build e2e

package e2e_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"
)

// installer is the script whose pin the test checks.
const installer = "../../install.sh"

// pinned reads a value install.sh pins, by its variable's name.
func pinned(t *testing.T, script []byte, name string) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^` + name + `="([^"]+)"$`).FindSubmatch(script)
	// Stop when the script pins no such value
	if m == nil {
		t.Fatalf("install.sh pins no %s", name)
	}
	return string(m[1])
}

// The devcontainer CLI's installer at the pinned tag still has the checksum install.sh pins, so a bump carries both.
func TestTheDevcontainersInstallerMatchesItsPin(t *testing.T) {
	script, err := os.ReadFile(installer)
	// Stop when the script fails to read
	if err != nil {
		t.Fatalf("ReadFile(install.sh) = %v", err)
	}
	version := pinned(t, script, "devcontainers_version")
	want := pinned(t, script, "devcontainers_installer_sha256")
	url := "https://raw.githubusercontent.com/devcontainers/cli/v" + version + "/scripts/install.sh"
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	// Stop when the request fails to build
	if err != nil {
		t.Fatalf("NewRequest = %v", err)
	}

	resp, err := http.DefaultClient.Do(req)

	// Stop when the installer fails to download
	if err != nil {
		t.Fatalf("GET %s = %v", url, err)
	}
	defer resp.Body.Close()
	// Stop when GitHub serves no installer at the tag
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %s", url, resp.Status)
	}
	h := sha256.New()
	// Stop when the download breaks off
	if _, err := io.Copy(h, resp.Body); err != nil {
		t.Fatalf("read %s = %v", url, err)
	}
	// Report a checksum that differs from the pin
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		t.Errorf("installer at v%s has sha256 %s, install.sh pins %s", version, got, want)
	}
}
