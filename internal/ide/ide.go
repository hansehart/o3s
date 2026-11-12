// Package ide opens an o3s home's workspace in VS Code straight in its cage.
package ide

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

// Env is the environment o3s runs in, as VS Code sees it.
type Env interface {
	// HostPath is the home's folder the way VS Code names it.
	HostPath(ctx context.Context, home string) (string, error)
	// Check names what VS Code needs before it opens the cage.
	Check(ctx context.Context) error
}

// Detect finds the environment o3s runs in, WSL where it names its distribution and Linux otherwise.
func Detect() Env {
	// Take WSL where it names its distribution
	if distro := os.Getenv("WSL_DISTRO_NAME"); distro != "" {
		return wsl{distro: distro}
	}
	return linux{}
}

// linux is a Linux host, where VS Code sees the folder as it is.
type linux struct{}

// HostPath hands back the folder as it is.
func (linux) HostPath(_ context.Context, home string) (string, error) {
	return home, nil
}

// Check passes, since VS Code runs Docker right on the host.
func (linux) Check(context.Context) error {
	return nil
}

// wsl is a WSL distribution, where VS Code runs on the Windows side.
type wsl struct {
	distro string
}

// HostPath hands back the folder's Windows form.
func (wsl) HostPath(ctx context.Context, home string) (string, error) {
	return output(ctx, "wslpath", "-w", home)
}

// Check passes where VS Code's settings let the Dev Containers extension run Docker in WSL, and names the lines to add otherwise.
func (w wsl) Check(ctx context.Context) error {
	path, err := settingsPath(ctx)
	// Pass where the settings hold the line
	if err == nil && executesInWSL(path) {
		return nil
	}
	return fmt.Errorf("VS Code runs Docker inside WSL with these lines in its settings.json:\n"+
		"  \"dev.containers.executeInWSL\": true,\n"+
		"  \"dev.containers.executeInWSLDistro\": %q", w.distro)
}

// settingsPath finds VS Code's user settings on the Windows side of WSL.
func settingsPath(ctx context.Context) (string, error) {
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

// executesInWSL reports whether VS Code's settings let the Dev Containers extension run Docker in WSL.
func executesInWSL(settingsPath string) bool {
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
