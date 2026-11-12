package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// running is the release the binary under test runs, and latest the one the releases page offers.
const running, latest = "0.7.0", "0.8.0"

// newBinary stands in for the o3s binary of the latest release.
const newBinary = "#!/bin/sh\necho o3s version " + latest + "\n"

// aReleasePage serves releases whose newest is the latest release.
func aReleasePage(t *testing.T) {
	t.Helper()
	sum := sha256.Sum256([]byte(newBinary))
	asset := "o3s-linux-" + runtime.GOARCH
	mux := http.NewServeMux()
	mux.HandleFunc("GET /latest", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/tag/v"+latest, http.StatusFound)
	})
	mux.HandleFunc("GET /download/v"+latest+"/"+asset, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(newBinary))
	})
	mux.HandleFunc("GET /download/v"+latest+"/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(hex.EncodeToString(sum[:]) + "  " + asset + "\n"))
	})
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	t.Setenv("O3S_RELEASES_URL", s.URL)
}

// aRunningBinary plays the running binary at a release, at a path the test owns, and records the restart.
func aRunningBinary(t *testing.T, at string) (string, *[]string) {
	t.Helper()
	old := Version
	Version = at
	t.Cleanup(func() { Version = old })
	path := filepath.Join(t.TempDir(), "o3s")
	// Stop when the running binary stays unwritten
	if err := os.WriteFile(path, []byte("old"), 0o755); err != nil { //nolint:gosec // the test plays a binary
		t.Fatalf("Setup: %v", err)
	}
	var restarted []string
	oldExecutable, oldRestart := executable, restartAs
	t.Cleanup(func() { executable, restartAs = oldExecutable, oldRestart })
	executable = func() (string, error) { return path, nil }
	restartAs = func(_ string, args []string) error { restarted = args; return nil }
	t.Setenv("PATH", t.TempDir())
	return path, &restarted
}

// A binary of an older release replaces itself with the latest and hands the rest of the update to it.
func TestSelfUpdateReplacesAnOlderRelease(t *testing.T) {
	aReleasePage(t)
	path, restarted := aRunningBinary(t, running)
	var out strings.Builder

	done, err := selfUpdate(t.Context(), strings.NewReader(""), &out, true)

	// Stop when the update fails
	if err != nil || !done {
		t.Fatalf("selfUpdate() = %v, %v, want it done\n%s", done, err, out.String())
	}
	data, _ := os.ReadFile(path)
	// Require the new release in place
	if string(data) != newBinary {
		t.Errorf("binary after selfUpdate = %q, want the new release", data)
	}
	// Require the new binary to take over the update
	if want := []string{path, "update", "--yes"}; !slices.Equal(*restarted, want) {
		t.Errorf("restarted as %q, want %q", *restarted, want)
	}
	// Require the version line, the confirmation and the hint for checking by hand
	for _, want := range []string{"o3s            " + running + " → " + latest, "Updated o3s to " + latest, "gh attestation verify " + path} {
		// Report a line the output lacks
		if !strings.Contains(out.String(), want) {
			t.Errorf("selfUpdate() output lacks %q:\n%s", want, out.String())
		}
	}
}

// A binary of the latest release, a dev build, and releases out of reach all go on to the home.
func TestSelfUpdateGoesOnToTheHome(t *testing.T) {
	tests := []struct {
		name, version, note string
		reachable           bool
	}{
		{"the latest release", latest, "", true},
		{"a dev build", "dev", "o3s dev builds update with go install or install.sh", true},
		{"releases out of reach", running, "⚠ releases out of reach, o3s stays at " + running, false},
	}
	// Check every case
	for _, tt := range tests {
		aReleasePage(t)
		// Point the releases at a closed port where the case plays them out of reach
		if !tt.reachable {
			t.Setenv("O3S_RELEASES_URL", "http://127.0.0.1:1")
		}
		_, restarted := aRunningBinary(t, tt.version)
		var out strings.Builder

		done, err := selfUpdate(t.Context(), strings.NewReader(""), &out, true)

		// Require the update to go on with this binary
		if err != nil || done || *restarted != nil {
			t.Errorf("%s: selfUpdate() = %v, %v, restarted %q, want to go on", tt.name, done, err, *restarted)
		}
		// Require the note the case gives, or none
		if strings.TrimSpace(out.String()) != tt.note {
			t.Errorf("%s: selfUpdate() output = %q, want %q", tt.name, out.String(), tt.note)
		}
	}
}
