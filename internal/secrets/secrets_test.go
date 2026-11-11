package secrets_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/secrets"
)

// header is the comment block the seeded file opens with.
const header = "# NAME\n#        secrets.env — real API tokens for the egress proxy\n"

// aSecretsFile hands a test a secrets file holding the given content, and its path.
func aSecretsFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "o3s", secrets.File)
	// Stop when the folder stays uncreated
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Stop when the file stays unwritten
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	return path
}

// load reads a secrets file, stopping the test when it fails.
func load(t *testing.T, path string) secrets.Store {
	t.Helper()
	s, err := secrets.Load(path)
	// Stop when the file stays unread
	if err != nil {
		t.Fatalf("Load(%q) = %v", path, err)
	}
	return s
}

// Reading follows the proxy, the last set value winning.
func TestValueReadsLikeTheProxy(t *testing.T) {
	path := aSecretsFile(t, header+"\nEXAMPLE_TOKEN = \"first\"\nEXAMPLE_TOKEN='second'\nEXAMPLE_TOKEN=\nBROKEN=a\rb\n")
	s := load(t, path)

	got, ok := s.Value("EXAMPLE_TOKEN")
	broken, brokenOK := s.Value("BROKEN")

	// Require the last value that holds anything
	if !ok || got != "second" {
		t.Errorf("Value(EXAMPLE_TOKEN) = %q, %v, want %q, true", got, ok, "second")
	}
	// Require a value with a line break to count as none
	if brokenOK || broken != "" {
		t.Errorf("Value(BROKEN) = %q, %v, want none", broken, brokenOK)
	}
}

// Setting a name fills the empty slot the start leaves, and keeps every other line as it was.
func TestSetFillsTheEmptySlot(t *testing.T) {
	path := aSecretsFile(t, header+"EXAMPLE_TOKEN=\n")
	s := load(t, path)

	replaced, err := s.Set("EXAMPLE_TOKEN", "s3cr3t")

	// Stop when the value is turned away
	if err != nil {
		t.Fatalf("Set(EXAMPLE_TOKEN) = %v", err)
	}
	// Require an empty slot to count as no earlier value
	if replaced {
		t.Errorf("Set(EXAMPLE_TOKEN) replaced = true, want false for an empty slot")
	}
	// Require the slot filled and the header kept
	if diff := cmp.Diff(header+"EXAMPLE_TOKEN=s3cr3t\n", s.String()); diff != "" {
		t.Errorf("Set(EXAMPLE_TOKEN) file (-want +got):\n%s", diff)
	}
}

// Setting a name that holds a value replaces it, drops later lines that could shadow it, and says so.
func TestSetReplacesAValueAndItsShadows(t *testing.T) {
	path := aSecretsFile(t, header+"EXAMPLE_TOKEN=old\nEXAMPLE_TOKEN=older\n")
	s := load(t, path)

	replaced, err := s.Set("EXAMPLE_TOKEN", "new")

	// Stop when the value is turned away
	if err != nil {
		t.Fatalf("Set(EXAMPLE_TOKEN) = %v", err)
	}
	// Require the rotation reported
	if !replaced {
		t.Errorf("Set(EXAMPLE_TOKEN) replaced = false, want true")
	}
	// Require one line holding the new value
	if diff := cmp.Diff(header+"EXAMPLE_TOKEN=new\n", s.String()); diff != "" {
		t.Errorf("Set(EXAMPLE_TOKEN) file (-want +got):\n%s", diff)
	}
}

// Setting a name the file lacks appends it.
func TestSetAppendsANewName(t *testing.T) {
	path := aSecretsFile(t, header)
	s := load(t, path)

	_, err := s.Set("EXAMPLE_TOKEN", "s3cr3t")

	// Stop when the value is turned away
	if err != nil {
		t.Fatalf("Set(EXAMPLE_TOKEN) = %v", err)
	}
	// Require the name after the header
	if diff := cmp.Diff(header+"EXAMPLE_TOKEN=s3cr3t\n", s.String()); diff != "" {
		t.Errorf("Set(EXAMPLE_TOKEN) file (-want +got):\n%s", diff)
	}
}

// Setting a value the proxy would refuse, or a name outside the rules, is turned away.
func TestSetTurnsAwayWhatTheProxyRefuses(t *testing.T) {
	tests := []struct {
		name, secret, value string
	}{
		{name: "empty value", secret: "EXAMPLE_TOKEN", value: "  "},
		{name: "line break", secret: "EXAMPLE_TOKEN", value: "a\nb"},
		{name: "carriage return", secret: "EXAMPLE_TOKEN", value: "a\rb"},
		{name: "lowercase name", secret: "example_token", value: "s3cr3t"},
	}
	// Check every case against its own empty file
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := load(t, aSecretsFile(t, header))

			_, err := s.Set(tt.secret, tt.value)

			// Require the rule named
			if !errors.Is(err, secrets.ErrInvalid) {
				t.Errorf("Set(%q, %q) = %v, want ErrInvalid", tt.secret, tt.value, err)
			}
		})
	}
}

// Removing a name drops every line of it and keeps the rest.
func TestRemoveDropsEveryLineOfAName(t *testing.T) {
	path := aSecretsFile(t, header+"EXAMPLE_TOKEN=a\nOTHER_TOKEN=b\nEXAMPLE_TOKEN=\n")
	s := load(t, path)

	ok := s.Remove("EXAMPLE_TOKEN")

	// Require the name gone and the rest kept
	if diff := cmp.Diff(header+"OTHER_TOKEN=b\n", s.String()); !ok || diff != "" {
		t.Errorf("Remove(EXAMPLE_TOKEN) = %v, file (-want +got):\n%s", ok, diff)
	}
}

// Names lists every name the file holds, with or without a value, in order.
func TestNamesListsEveryName(t *testing.T) {
	s := load(t, aSecretsFile(t, header+"OTHER_TOKEN=b\nEXAMPLE_TOKEN=\n"))

	got := s.Names()

	// Require both names, sorted
	if diff := cmp.Diff([]string{"EXAMPLE_TOKEN", "OTHER_TOKEN"}, got); diff != "" {
		t.Errorf("Names() (-want +got):\n%s", diff)
	}
}

// Saving writes the file for its owner alone, in a folder for its owner alone, and reads back the same.
func TestSaveKeepsTheFileToItsOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "o3s", secrets.File)
	s := load(t, path)
	// Stop when the value is turned away
	if _, err := s.Set("EXAMPLE_TOKEN", "s3cr3t"); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	err := s.Save(path)

	// Stop when the file stays unwritten
	if err != nil {
		t.Fatalf("Save(%q) = %v", path, err)
	}
	file, _ := os.Stat(path)
	dir, _ := os.Stat(filepath.Dir(path))
	// Require the owner alone to read the file and its folder
	if file.Mode().Perm() != 0o600 || dir.Mode().Perm() != 0o700 {
		t.Errorf("Save(%q) modes = file %v, folder %v, want 0600 and 0700", path, file.Mode().Perm(), dir.Mode().Perm())
	}
	got, _ := load(t, path).Value("EXAMPLE_TOKEN")
	// Require the value to read back
	if got != "s3cr3t" {
		t.Errorf("Value after Save = %q, want %q", got, "s3cr3t")
	}
}
