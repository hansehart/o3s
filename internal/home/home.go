// Package home makes a folder into an o3s home.
package home

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"

	"github.com/hansehart/o3s/internal/atomicfile"
	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/release"
)

// ErrExists marks a folder that already holds a home.
var ErrExists = errors.New("already belongs to an o3s home")

// stackDir is where a home keeps its stack, the dev container tooling's own folder.
const stackDir = ".devcontainer"

// Version is the version of the running binary, whose release tag the stack's own images take.
var Version = "dev"

// Executable is the running binary's own path, which a release writes as the stack's start hook.
var Executable string

// startHook is the start hook the shipped config names, which a release replaces with its own path.
const startHook = `"initializeCommand": "o3s init"`

// images are the images o3s publishes with each release.
var images = []string{"ghcr.io/hansehart/o3s-base", "ghcr.io/hansehart/o3s-gateway"}

// Init writes a new home into a folder and hands back the paths it wrote.
func Init(dir string) ([]string, error) {
	// Turn away a folder that already holds a home, before anything is written
	for _, name := range []string{config.File, stackDir} {
		// Name the first part of a home the folder holds
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return nil, fmt.Errorf("%s %w", filepath.Join(dir, name), ErrExists)
		}
	}
	settings := config.New()
	// Hand back settings that stay unwritten
	if err := settings.Save(dir); err != nil {
		return nil, err
	}
	written := []string{filepath.Join(dir, config.File)}
	files, err := stack()
	// Hand back a stack the binary fails to hand over
	if err != nil {
		return written, err
	}
	// Write every file of the stack
	for _, f := range files {
		target := filepath.Join(dir, stackDir, filepath.FromSlash(f.path))
		// Hand back a file that stays unwritten
		if err := writeFile(target, f.data); err != nil {
			return written, err
		}
		written = append(written, target)
	}
	return written, nil
}

// file is one file of the stack, its path inside the stack folder.
type file struct {
	path string
	data []byte
}

// stack hands back every file of the stack a home holds, its config without features.
func stack() ([]file, error) {
	var stack []file
	err := fs.WalkDir(files, source, func(path string, d fs.DirEntry, err error) error {
		// Hand back a walk that fails, and take folders as they come with their files
		if err != nil || d.IsDir() {
			return err
		}
		data, err := files.ReadFile(path)
		// Hand back a file the binary fails to hand over
		if err != nil {
			return err
		}
		// Start the config with no feature, the way a home starts with no host, and the binary as its start hook
		if d.Name() == "devcontainer.json" {
			data, err = withoutFeatures(data)
			// Hand back a config that fails to edit
			if err != nil {
				return err
			}
			data = hooked(data)
		}
		// Run the images of the release the binary belongs to
		if d.Name() == "compose.yaml" {
			data = pinned(data)
		}
		stack = append(stack, file{path: strings.TrimPrefix(path, source+"/"), data: data})
		return nil
	})
	return stack, err
}

// hooked names the release binary's own path as the start hook, so the editor finds it whatever its PATH holds.
func hooked(config []byte) []byte {
	// Keep the plain command for a build that names no release
	if !release.IsRelease(Version) || Executable == "" {
		return config
	}
	path, _ := json.Marshal(Executable)
	return bytes.Replace(config, []byte(startHook), []byte(`"initializeCommand": [`+string(path)+`, "init"]`), 1)
}

// Exists reports whether a folder holds an o3s home, its settings or its stack.
func Exists(dir string) bool {
	// Look for either part of a home
	for _, name := range []string{config.File, stackDir} {
		// Report the first part the folder holds
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// Seed writes the files a home needs and lacks, its settings and its workspace named after the folder, and hands back what it wrote.
func Seed(dir string) ([]string, error) {
	var written []string
	settings := filepath.Join(dir, config.File)
	// Start settings a home lost with the ones a new home gets
	if _, err := os.Stat(settings); errors.Is(err, fs.ErrNotExist) {
		c := config.New()
		// Hand back settings that stay unwritten
		if err := c.Save(dir); err != nil {
			return written, err
		}
		written = append(written, settings)
	}
	workspace := filepath.Join(dir, filepath.Base(dir)+".code-workspace")
	// Start the workspace from its template where the home holds none
	if _, err := os.Stat(workspace); errors.Is(err, fs.ErrNotExist) {
		data, err := files.ReadFile("templates/o3s.code-workspace")
		// Hand back a template the binary fails to hand over
		if err != nil {
			return written, err
		}
		// Hand back a workspace that stays unwritten
		if err := writeFile(workspace, data); err != nil {
			return written, err
		}
		written = append(written, workspace)
	}
	return written, nil
}

// pinned tags o3s's own images with the binary's release, and leaves latest for any other build.
func pinned(compose []byte) []byte {
	// Keep latest for a build that names no release
	if !release.IsRelease(Version) {
		return compose
	}
	// Tag every o3s image with the release
	for _, image := range images {
		compose = bytes.ReplaceAll(compose, []byte(image+":latest"), []byte(image+":"+Version))
	}
	return compose
}

// writeFile writes a file of the stack in a single step, with the folders it needs.
func writeFile(path string, data []byte) error {
	// Hand back folders that stay uncreated
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // the stack's folders are read by the containers
		return err
	}
	return atomicfile.Write(path, data, 0o644)
}

// withoutFeatures empties a config's features and keeps everything else.
func withoutFeatures(data []byte) ([]byte, error) {
	v, err := hujson.Parse(data)
	// Hand back a config that holds no JSON with comments
	if err != nil {
		return nil, err
	}
	// Hand back a config whose features stay as they were
	if err := v.Patch([]byte(`[{"op": "replace", "path": "/features", "value": {}}]`)); err != nil {
		return nil, err
	}
	return v.Pack(), nil
}

// SeedSecrets starts the secrets file from its template where the user holds none, in a folder for the user alone.
func SeedSecrets(path string) (bool, error) {
	// Keep a secrets file the user holds
	if _, err := os.Stat(path); err == nil {
		return false, nil
	}
	data, err := files.ReadFile("templates/secrets.env")
	// Hand back a template the binary fails to hand over
	if err != nil {
		return false, err
	}
	// Hand back a folder that stays uncreated
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, err
	}
	return true, atomicfile.Write(path, data, 0o600)
}

// Holds reports the first of the host's folders a home would hold, which the cage would then see.
func Holds(dir string, folders ...string) (string, bool) {
	// Look at every folder of the host
	for _, f := range folders {
		rel, err := filepath.Rel(dir, f)
		// Name a folder at or below the home
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return f, true
		}
	}
	return "", false
}
