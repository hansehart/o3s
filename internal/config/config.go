// Package config reads and writes the hosts and stores of an o3s home.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	// File is the name of the settings file in an o3s home.
	File = "o3s.json"
	// schema is where the settings file finds the schema o3s writes beside it.
	schema = "./.devcontainer/o3s.schema.json"
)

var (
	// ErrNoHome marks a folder that holds no o3s home.
	ErrNoHome = errors.New("no o3s home")
	// ErrInvalid marks a change that breaks one of the allowlist's rules.
	ErrInvalid = errors.New("invalid")
	// ErrUnknownHost marks a host the allowlist lacks.
	ErrUnknownHost = errors.New("unknown host")
	// ErrOwnedByFeatures marks a host that features alone brought, so removing them removes it.
	ErrOwnedByFeatures = errors.New("needed by features alone")
)

// user is the name in a host's needed-by list for what the user added by hand.
const user = "user"

// Config is what an o3s home holds, in the order o3s writes it.
type Config struct {
	Schema string  `json:"$schema"`
	Stores []Store `json:"stores"`
	Hosts  []Host  `json:"hosts"`
}

// Store is a feature collection the menu offers features from.
type Store struct {
	ID string `json:"id"`
}

// Host is one allowlist entry, where the cage may connect and what the proxy injects there.
type Host struct {
	Name   string   `json:"name"`
	Ports  []int    `json:"ports"`
	Secret string   `json:"secret,omitempty"`
	Inject []string `json:"inject,omitempty"`
	By     []string `json:"by"`
}

// Change names a host and the fields to set, an empty field keeping its value or default.
type Change struct {
	Name   string
	Ports  []int
	Secret string
	Inject []string
}

// New hands back the settings of a new home, with both feature stores and no host.
func New() Config {
	return Config{
		Stores: []Store{{ID: "ghcr.io/devcontainers/features"}, {ID: "ghcr.io/hansehart/devcontainer-features"}},
		Hosts:  []Host{},
	}
}

// Load reads the settings of the o3s home in a folder.
func Load(home string) (Config, error) {
	data, err := os.ReadFile(filepath.Join(home, File))
	// Point a folder without a home at the command that starts one
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("%w in %s, start one with o3s init", ErrNoHome, home)
	}
	// Hand back any other failure to read
	if err != nil {
		return Config{}, err
	}
	var c Config
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	// Turn away a file that holds anything but the settings o3s writes
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("read %s: %w", File, err)
	}
	return c, nil
}

// Save writes the settings in one stable shape and replaces the file in a single step.
func (c *Config) Save(home string) error {
	c.Schema = schema
	c.order()
	// Write empty lists as lists, so the shape stays the same
	if c.Stores == nil {
		c.Stores = []Store{}
	}
	// Write empty lists as lists, so the shape stays the same
	if c.Hosts == nil {
		c.Hosts = []Host{}
	}
	data, err := json.MarshalIndent(c, "", "  ")
	// Hand back a config that fails to encode
	if err != nil {
		return err
	}
	return replace(filepath.Join(home, File), append(data, '\n'))
}

// replace writes a file beside its target and renames it into place.
func replace(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	// Hand back a folder that takes no new file
	if err != nil {
		return err
	}
	// Clean up the temporary file whenever it stays behind
	defer os.Remove(tmp.Name())
	// Let the stack read the settings it enforces
	if err := tmp.Chmod(0o644); err != nil {
		return errors.Join(err, tmp.Close())
	}
	// Hand back a write that fails part way
	if _, err := tmp.Write(data); err != nil {
		return errors.Join(err, tmp.Close())
	}
	// Hand back a file that fails to close
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
