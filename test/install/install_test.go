// Package install_test runs install.sh against a release served from the test, on a host the test plays.
package install_test

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// script is the installer under test.
const script = "../../install.sh"

// stub stands in for the o3s binary of a release.
const stub = "#!/bin/sh\necho o3s version 0.7.0\n"

// tools are the commands install.sh may use beyond the ones a test fakes.
var tools = []string{"sh", "curl", "sha256sum", "mktemp", "install", "rm", "mkdir", "grep", "cat"}

// aRelease serves a release holding the stub for amd64, and checksums that match it unless tampered.
func aRelease(t *testing.T, tampered bool) string {
	t.Helper()
	dir := t.TempDir()
	sum := sha256.Sum256([]byte(stub))
	digest := hex.EncodeToString(sum[:])
	// Break the checksum where the test plays a tampered download
	if tampered {
		digest = strings.Repeat("0", 64)
	}
	files := map[string]string{
		"o3s-linux-amd64": stub,
		"checksums.txt":   digest + "  o3s-linux-amd64\n" + strings.Repeat("1", 64) + "  install.sh\n",
	}
	// Write every file of the release
	for name, body := range files {
		// Stop when a file stays unwritten
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("Setup: %v", err)
		}
	}
	s := httptest.NewServer(http.FileServer(http.Dir(dir)))
	t.Cleanup(s.Close)
	return s.URL
}

// host is a machine the test plays, with the tools install.sh may use and the fakes it adds.
type host struct {
	home, bin string
	env       []string
}

// aHost builds a host that reports a kernel and an architecture, and runs the fake commands given.
func aHost(t *testing.T, kernel, arch string, fakes map[string]string) host {
	t.Helper()
	h := host{home: t.TempDir(), bin: t.TempDir()}
	// Link every tool the script may use from this machine
	for _, tool := range tools {
		path, err := exec.LookPath(tool)
		// Stop when this machine lacks a tool the script needs
		if err != nil {
			t.Fatalf("Setup: %s: %v", tool, err)
		}
		// Stop when the tool fails to link
		if err := os.Symlink(path, filepath.Join(h.bin, tool)); err != nil {
			t.Fatalf("Setup: %v", err)
		}
	}
	fakes["uname"] = "case \"$1\" in -s) echo " + kernel + " ;; -m) echo " + arch + " ;; esac\n"
	// Write every fake command
	for name, body := range fakes {
		// Stop when a fake stays unwritten
		if err := os.WriteFile(filepath.Join(h.bin, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
			t.Fatalf("Setup: %v", err)
		}
	}
	h.env = []string{"HOME=" + h.home, "PATH=" + h.bin}
	return h
}

// install runs install.sh on the host against a release, without the devcontainer CLI.
func (h host) install(t *testing.T, release string) (string, error) {
	t.Helper()
	return h.run(t, release, "--no-devcontainer")
}

// run runs install.sh on the host against a release with the options given.
func (h host) run(t *testing.T, release string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "sh", append([]string{script}, args...)...)
	cmd.Env = append(slices.Clone(h.env), "O3S_RELEASE_URL="+release)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// installed reports whether the host holds an o3s binary in its install dir.
func (h host) installed() bool {
	info, err := os.Stat(filepath.Join(h.home, ".local", "bin", "o3s"))
	return err == nil && info.Mode()&0o111 != 0
}

// The binary lands executable in ~/.local/bin, with hints for PATH and for verifying it by hand.
func TestInstallPutsTheBinaryInPlace(t *testing.T) {
	h := aHost(t, "Linux", "x86_64", map[string]string{})

	out, err := h.install(t, aRelease(t, false))

	// Stop when the install fails
	if err != nil {
		t.Fatalf("install.sh = %v\n%s", err, out)
	}
	// Require the binary in place
	if !h.installed() {
		t.Errorf("install.sh left no executable o3s in ~/.local/bin\n%s", out)
	}
	// Require every line a user needs next
	for _, want := range []string{"Installed o3s version 0.7.0", "export PATH=", "gh attestation verify", "--signer-workflow hansehart/o3s/.github/workflows/publish.yaml", "o3s init"} {
		// Report a line the output lacks
		if !strings.Contains(out, want) {
			t.Errorf("install.sh output lacks %q:\n%s", want, out)
		}
	}
}

// A download whose checksum differs stops the install before anything is written.
func TestInstallStopsOnATamperedDownload(t *testing.T) {
	h := aHost(t, "Linux", "x86_64", map[string]string{})

	out, err := h.install(t, aRelease(t, true))

	// Require the install to stop
	if err == nil || h.installed() {
		t.Errorf("install.sh on a tampered download = %v, installed %v, want a stop\n%s", err, h.installed(), out)
	}
	// Require the reason
	if !strings.Contains(out, "checksums.txt") {
		t.Errorf("install.sh output lacks the checksum reason:\n%s", out)
	}
}

// A host o3s ships no binary for is named and turned away.
func TestInstallTurnsAwayAnUnsupportedHost(t *testing.T) {
	tests := []struct{ kernel, arch, want string }{
		{"Linux", "riscv64", "riscv64"},
		{"Darwin", "arm64", "Darwin"},
	}
	// Check every host
	for _, tt := range tests {
		h := aHost(t, tt.kernel, tt.arch, map[string]string{})

		out, err := h.install(t, aRelease(t, false))

		// Require a stop that names the host
		if err == nil || !strings.Contains(out, tt.want) {
			t.Errorf("install.sh on %s %s = %v, want a stop naming %s\n%s", tt.kernel, tt.arch, err, tt.want, out)
		}
	}
}

// A logged-in gh verifies the attestation against this repo's publish workflow, and a failed check stops the install.
func TestInstallVerifiesTheAttestationWithGh(t *testing.T) {
	tests := []struct {
		name   string
		verify string
		ok     bool
	}{
		{"a verified build", "exit 0", true},
		{"a build that fails to verify", "exit 1", false},
	}
	// Check every outcome
	for _, tt := range tests {
		gh := "echo \"$*\" >> \"$HOME/gh.log\"\ncase \"$1\" in auth) exit 0 ;; attestation) " + tt.verify + " ;; esac\n"
		h := aHost(t, "Linux", "x86_64", map[string]string{"gh": gh})

		out, err := h.install(t, aRelease(t, false))

		// Require the outcome gh decides
		if (err == nil) != tt.ok || h.installed() != tt.ok {
			t.Errorf("%s: install.sh = %v, installed %v, want ok %v\n%s", tt.name, err, h.installed(), tt.ok, out)
		}
		calls, _ := os.ReadFile(filepath.Join(h.home, "gh.log"))
		want := "attestation verify o3s-linux-amd64 --repo hansehart/o3s --signer-workflow hansehart/o3s/.github/workflows/publish.yaml"
		// Require gh asked about this repo's publish workflow
		if !strings.Contains(string(calls), want) {
			t.Errorf("%s: gh calls lack %q:\n%s", tt.name, want, calls)
		}
	}
}

// A host without xz, which the devcontainer CLI's installer unpacks Node with, is told what to do before anything installs.
func TestInstallNamesTheMissingXz(t *testing.T) {
	h := aHost(t, "Linux", "x86_64", map[string]string{})

	out, err := h.run(t, aRelease(t, false))

	// Require a stop that names xz and the way around it
	if err == nil || !strings.Contains(out, "xz") || !strings.Contains(out, "--no-devcontainer") {
		t.Errorf("install.sh without xz = %v, want a stop naming xz and --no-devcontainer\n%s", err, out)
	}
	// Require the host left as it was, so one run after installing xz does it all
	if h.installed() {
		t.Errorf("install.sh without xz installed o3s anyway")
	}
}

// The workflow install.sh accepts as the signer lives in the repo, so a rename fails here before it fails every install.
func TestTheSignerWorkflowExists(t *testing.T) {
	data, err := os.ReadFile(script)
	// Stop when the script fails to read
	if err != nil {
		t.Fatalf("ReadFile(install.sh) = %v", err)
	}
	m := regexp.MustCompile(`(?m)^signer="\$\{repo\}/(.+)"$`).FindSubmatch(data)
	// Stop when the script pins no signer
	if m == nil {
		t.Fatalf("install.sh pins no signer")
	}

	_, err = os.Stat(filepath.Join("../..", string(m[1])))

	// Report a signer the repo lacks
	if err != nil {
		t.Errorf("signer %s: %v", m[1], err)
	}
}
