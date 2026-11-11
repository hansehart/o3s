// Package secrets reads and writes the values the proxy injects, keeping every other line.
package secrets

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hansehart/o3s/internal/atomicfile"
	"github.com/hansehart/o3s/internal/config"
)

// File is the name of the secrets file.
const File = "secrets.env"

var (
	// ErrInvalid marks a name or value the proxy would refuse.
	ErrInvalid = errors.New("invalid")
	// ErrUnknown marks a name the file lacks.
	ErrUnknown = errors.New("unknown secret")
)

// Store is the secrets file, line by line.
type Store struct {
	lines []string
}

// Path finds the secrets file the stack mounts into the proxy.
func Path() (string, error) {
	// Follow the folder the stack is told to use
	if dir := os.Getenv("O3S_SECRETS_DIR"); dir != "" {
		return filepath.Join(dir, File), nil
	}
	home, err := os.UserHomeDir()
	// Hand back a home folder that stays unknown
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "o3s", File), nil
}

// Load reads a secrets file, and hands back an empty store for a file that is yet to exist.
func Load(path string) (Store, error) {
	data, err := os.ReadFile(path)
	// Start empty where the file is yet to exist
	if errors.Is(err, fs.ErrNotExist) {
		return Store{}, nil
	}
	// Hand back any other failure to read
	if err != nil {
		return Store{}, err
	}
	text := strings.TrimSuffix(string(data), "\n")
	// Keep an empty file empty
	if text == "" {
		return Store{}, nil
	}
	return Store{lines: strings.Split(text, "\n")}, nil
}

// Value returns a secret's value the way the proxy reads it, the last set line winning.
func (s Store) Value(name string) (string, bool) {
	var value string
	// Read every line, keeping the last value of the name
	for _, line := range s.lines {
		n, v, ok := parse(line)
		// Keep a value of the name that holds anything
		if ok && n == name && v != "" {
			value = v
		}
	}
	return value, value != ""
}

// Names lists every name the file holds, with or without a value, in order.
func (s Store) Names() []string {
	var names []string
	// Collect the name of every entry line
	for _, line := range s.lines {
		// Keep the name of a line that holds one
		if n, _, ok := parse(line); ok && !slices.Contains(names, n) {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	return names
}

// Set stores a secret so no older value shadows it, and reports whether it replaced one.
func (s *Store) Set(name, value string) (bool, error) {
	value = strings.TrimSpace(value)
	// Turn away a name the proxy reads under no variable
	if !config.IsSecretName(name) {
		return false, fmt.Errorf("%w secret %q, want an uppercase env name", ErrInvalid, name)
	}
	// Turn away a value the proxy reads as none
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return false, fmt.Errorf("%w value for %s, want one line that holds something", ErrInvalid, name)
	}
	_, replaced := s.Value(name)
	entry := name + "=" + value
	var lines []string
	stored := false
	// Rewrite the lines of the name, keeping every other line as it was
	for _, line := range s.lines {
		n, _, ok := parse(line)
		// Keep a line of another name, or any comment
		if !ok || n != name {
			lines = append(lines, line)
			continue
		}
		// Put the value on the first line of the name, and drop the rest
		if !stored {
			lines = append(lines, entry)
			stored = true
		}
	}
	// Append a name the file lacks
	if !stored {
		lines = append(lines, entry)
	}
	s.lines = lines
	return replaced, nil
}

// Remove drops every line of a name, and reports whether the file held it.
func (s *Store) Remove(name string) bool {
	before := len(s.lines)
	s.lines = slices.DeleteFunc(s.lines, func(line string) bool {
		n, _, ok := parse(line)
		return ok && n == name
	})
	return len(s.lines) < before
}

// String hands back the file as it would be written.
func (s Store) String() string {
	// Write an empty store as an empty file
	if len(s.lines) == 0 {
		return ""
	}
	return strings.Join(s.lines, "\n") + "\n"
}

// Save writes the file for its owner alone and replaces it in a single step.
func (s Store) Save(path string) error {
	// Hand back a folder that stays uncreated
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return atomicfile.Write(path, []byte(s.String()), 0o600)
}

// parse reads one line the way the proxy does.
func parse(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	// Skip a line that holds no entry
	if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
		return "", "", false
	}
	name, value, _ := strings.Cut(line, "=")
	value = strings.Trim(strings.Trim(strings.TrimSpace(value), `"`), `'`)
	// Read a value with a line break as none
	if strings.ContainsAny(value, "\r\n") {
		value = ""
	}
	return strings.TrimSpace(name), value, true
}
