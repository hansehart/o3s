// Package ide opens an o3s home's workspace in VS Code straight in its cage.
package ide

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/tailscale/hujson"
)

// workspaceFolder is where the cage mounts the home, as the stack's devcontainer.json names it.
const workspaceFolder = "/home/ubuntu/o3s"

// URI is the address VS Code opens a home's workspace at, the host's folder in hex and the workspace inside the cage.
func URI(hostPath, workspace string) string {
	return "vscode-remote://dev-container+" + hex.EncodeToString([]byte(hostPath)) + path.Join(workspaceFolder, workspace)
}

// WSL reports whether o3s runs inside WSL, and the distribution it runs in.
func WSL() (string, bool) {
	distro := os.Getenv("WSL_DISTRO_NAME")
	return distro, distro != ""
}

// HostPath is the home's folder the way VS Code names it, its Windows form under WSL.
func HostPath(ctx context.Context, home string) (string, error) {
	// Keep the folder as it is outside WSL
	if _, ok := WSL(); !ok {
		return home, nil
	}
	return output(ctx, "wslpath", "-w", home)
}

// SettingsPath finds VS Code's user settings on the Windows side of WSL.
func SettingsPath(ctx context.Context) (string, error) {
	appData, err := output(ctx, "cmd.exe", "/c", "echo %APPDATA%")
	// Hand back a Windows folder that stays unknown
	if err != nil {
		return "", err
	}
	dir, err := output(ctx, "wslpath", "-u", appData)
	// Hand back a Windows folder WSL fails to name
	if err != nil {
		return "", err
	}
	return path.Join(dir, "Code", "User", "settings.json"), nil
}

// ExecutesInWSL reports whether VS Code's settings let the Dev Containers extension run Docker in WSL.
func ExecutesInWSL(settingsPath string) bool {
	data, err := os.ReadFile(settingsPath)
	// Read settings that fail to read as unset
	if err != nil {
		return false
	}
	plain, err := hujson.Standardize(data)
	// Read settings that fail to parse as unset
	if err != nil {
		return false
	}
	var settings map[string]any
	// Read settings in another shape as unset
	if json.Unmarshal(plain, &settings) != nil {
		return false
	}
	on, _ := settings["dev.containers.executeInWSL"].(bool)
	return on
}

// output runs a command and hands back what it printed, trimmed of line ends.
func output(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return strings.TrimSpace(string(out)), err
}
