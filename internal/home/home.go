// Package home makes a folder into an o3s home.
package home

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tailscale/hujson"

	"github.com/hansehart/o3s"
	"github.com/hansehart/o3s/internal/config"
)

// ErrExists marks a folder that already holds a home.
var ErrExists = errors.New("already belongs to an o3s home")

// stackDir is where a home keeps its stack.
const stackDir = ".devcontainer"

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
	err := fs.WalkDir(o3s.Stack, stackDir, func(path string, d fs.DirEntry, err error) error {
		// Hand back a walk that fails
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(path))
		// Create every folder of the stack
		if d.IsDir() {
			return os.MkdirAll(target, 0o755) //nolint:gosec // the stack's folders are read by the containers
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
		written = append(written, target)
		return os.WriteFile(target, data, 0o644) //nolint:gosec // the stack's files are read by the containers
	})
	return written, err
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
