package ide_test

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hansehart/o3s/internal/ide"
)

// The URI names the host's folder in hex and the workspace inside the cage, as VS Code opens it.
func TestURIOpensTheWorkspaceInTheCage(t *testing.T) {
	tests := []struct{ host, want string }{
		{"/home/ubuntu/o3s", "vscode-remote://dev-container+" + hex.EncodeToString([]byte("/home/ubuntu/o3s")) + "/home/ubuntu/o3s/o3s.code-workspace"},
		{`\\wsl.localhost\Ubuntu\home\ubuntu\o3s`, "vscode-remote://dev-container+" + hex.EncodeToString([]byte(`\\wsl.localhost\Ubuntu\home\ubuntu\o3s`)) + "/home/ubuntu/o3s/o3s.code-workspace"},
	}
	// Check every host path
	for _, tt := range tests {
		// Report a URI that differs
		if got := ide.URI(tt.host, "o3s.code-workspace"); got != tt.want {
			t.Errorf("URI(%q) = %q, want %q", tt.host, got, tt.want)
		}
	}
}

// fakes puts commands on PATH, each a shell script the test gives.
func fakes(t *testing.T, scripts map[string]string) {
	t.Helper()
	bin := t.TempDir()
	// Write every fake
	for name, script := range scripts {
		// Stop when the fake stays unwritten
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil { //nolint:gosec // the test runs its fakes
			t.Fatalf("Setup: %v", err)
		}
	}
	t.Setenv("PATH", bin)
}

// aWindows fakes WSL's Windows side, whose VS Code keeps the given settings, or none where they are empty.
func aWindows(t *testing.T, settings string) {
	t.Helper()
	appData := t.TempDir()
	fakes(t, map[string]string{
		"cmd.exe": `echo 'C:\Users\ubuntu\AppData\Roaming'`,
		"wslpath": `[ "$1" = -u ] && echo '` + appData + `'`,
	})
	// Leave VS Code without settings where the test gives none
	if settings == "" {
		return
	}
	dir := filepath.Join(appData, "Code", "User")
	// Stop when the settings stay unwritten
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Stop when the settings stay unwritten
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(settings), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}
}

// Under WSL the host path is the folder's Windows form, which the Dev Containers extension reads.
func TestHostPathTakesTheWindowsFormUnderWSL(t *testing.T) {
	fakes(t, map[string]string{"wslpath": `[ "$1" = -w ] && printf '%s\n' '\\wsl.localhost\Ubuntu'"$2"`})
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")

	got, err := ide.Detect().HostPath(t.Context(), "/home/ubuntu/o3s")

	// Report a path that differs
	if want := `\\wsl.localhost\Ubuntu/home/ubuntu/o3s`; err != nil || got != want {
		t.Errorf("HostPath() = %q, %v, want %q", got, err, want)
	}
}

// Outside WSL the host path is the folder as it is, and VS Code needs nothing first.
func TestLinuxKeepsTheFolderAndNeedsNothing(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "")
	env := ide.Detect()

	got, err := env.HostPath(t.Context(), "/home/ubuntu/o3s")

	// Report a path that differs
	if err != nil || got != "/home/ubuntu/o3s" {
		t.Errorf("HostPath() = %q, %v, want the folder itself", got, err)
	}
	// Report a check that asks for something
	if err := env.Check(t.Context()); err != nil {
		t.Errorf("Check() = %v, want nil", err)
	}
}

// Under WSL the check passes where VS Code's settings set executeInWSL true, and names both lines otherwise.
func TestWSLCheckReadsTheSetting(t *testing.T) {
	tests := []struct {
		name, settings string
		ok             bool
	}{
		{"set true, with comments", "{\n  // run the extension's docker in WSL\n  \"dev.containers.executeInWSL\": true,\n}\n", true},
		{"set false", `{"dev.containers.executeInWSL": false}`, false},
		{"missing", `{"editor.fontSize": 14}`, false},
		{"no settings", "", false},
	}
	// Check every settings file
	for _, tt := range tests {
		aWindows(t, tt.settings)
		t.Setenv("WSL_DISTRO_NAME", "Ubuntu")

		err := ide.Detect().Check(t.Context())

		// Report a setting VS Code holds that the check turns away
		if tt.ok && err != nil {
			t.Errorf("%s: Check() = %v, want nil", tt.name, err)
		}
		// Report a missing setting the check lets through, or names without both lines
		if !tt.ok && (err == nil || !strings.Contains(err.Error(), `"dev.containers.executeInWSL": true`) || !strings.Contains(err.Error(), `"dev.containers.executeInWSLDistro": "Ubuntu"`)) {
			t.Errorf("%s: Check() = %v, want both lines named", tt.name, err)
		}
	}
}
