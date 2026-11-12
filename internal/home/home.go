// Package home makes a folder into an o3s home.
package home

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"

	"github.com/hansehart/o3s"
	"github.com/hansehart/o3s/internal/atomicfile"
	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/release"
)

// ErrExists marks a folder that already holds a home.
var ErrExists = errors.New("already belongs to an o3s home")

// stackDir is where a home keeps its stack.
const stackDir = ".devcontainer"

// Version is the version of the running binary, whose release tag the stack's own images take.
var Version = "dev"

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

// stack hands back every file of the stack the binary ships, its config without features.
func stack() ([]file, error) {
	var files []file
	err := fs.WalkDir(o3s.Stack, stackDir, func(path string, d fs.DirEntry, err error) error {
		// Hand back a walk that fails, and take folders as they come with their files
		if err != nil || d.IsDir() {
			return err
		}
		data, err := o3s.Stack.ReadFile(path)
		// Hand back a file the binary fails to hand over
		if err != nil {
			return err
		}
		// Start the config with no feature, the way a home starts with no host
		if d.Name() == "devcontainer.json" {
			data, err = withoutFeatures(data)
			// Hand back a config that fails to edit
			if err != nil {
				return err
			}
		}
		// Run the images of the release the binary belongs to
		if d.Name() == "compose.yaml" {
			data = pinned(data)
		}
		files = append(files, file{path: strings.TrimPrefix(path, stackDir+"/"), data: data})
		return nil
	})
	return files, err
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
