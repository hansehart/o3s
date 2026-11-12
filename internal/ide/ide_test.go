package ide_test

import (
	"encoding/hex"
	"os"
	"path/filepath"
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

// aFake puts a command on PATH that prints what the test gives it.
func aFake(t *testing.T, name, output string) {
	t.Helper()
	bin := t.TempDir()
	// Stop when the fake stays unwritten
	if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nprintf '%s\\n' '"+output+"'\n"), 0o755); err != nil { //nolint:gosec // the test runs its fake
		t.Fatalf("Setup: %v", err)
	}
	t.Setenv("PATH", bin)
}

// Under WSL the host path is the folder's Windows form, which the Dev Containers extension reads.
func TestHostPathTakesTheWindowsFormUnderWSL(t *testing.T) {
	aFake(t, "wslpath", `\\wsl.localhost\Ubuntu\home\ubuntu\o3s`)
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")

	got, err := ide.HostPath(t.Context(), "/home/ubuntu/o3s")

	// Report a path that differs
	if err != nil || got != `\\wsl.localhost\Ubuntu\home\ubuntu\o3s` {
		t.Errorf("HostPath() = %q, %v, want the Windows form", got, err)
	}
}

// Outside WSL the host path is the folder as it is.
func TestHostPathKeepsTheFolderOutsideWSL(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "")

	got, err := ide.HostPath(t.Context(), "/home/ubuntu/o3s")

	// Report a path that differs
	if err != nil || got != "/home/ubuntu/o3s" {
		t.Errorf("HostPath() = %q, %v, want the folder itself", got, err)
	}
}

// The setting counts when VS Code's settings set it true, comments and all, and in no other case.
func TestExecutesInWSLReadsTheSetting(t *testing.T) {
	tests := []struct {
		name, settings string
		want           bool
	}{
		{"set true, with comments", "{\n  // run the extension's docker in WSL\n  \"dev.containers.executeInWSL\": true,\n}\n", true},
		{"set false", `{"dev.containers.executeInWSL": false}`, false},
		{"missing", `{"editor.fontSize": 14}`, false},
	}
	// Check every settings file
	for _, tt := range tests {
		path := filepath.Join(t.TempDir(), "settings.json")
		// Stop when the settings stay unwritten
		if err := os.WriteFile(path, []byte(tt.settings), 0o600); err != nil {
			t.Fatalf("Setup: %v", err)
		}

		got := ide.ExecutesInWSL(path)

		// Report a reading that differs
		if got != tt.want {
			t.Errorf("%s: ExecutesInWSL() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
