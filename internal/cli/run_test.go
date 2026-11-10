package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hansehart/o3s/internal/cli"
	"github.com/hansehart/o3s/internal/config"
)

// aHome hands a test an o3s home with an empty allowlist, and works inside it.
func aHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	c := config.Config{}
	// Stop when the home stays unwritten
	if err := c.Save(home); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	t.Chdir(home)
}

// run hands a command line to the CLI and hands back its status and what it reported on stderr.
func run(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var out, errOut strings.Builder
	status := cli.Run(t.Context(), args, cli.Streams{In: strings.NewReader(""), Out: &out, Err: &errOut})
	return status, errOut.String()
}

// The exit status tells a usage error apart from a failure and from success.
func TestRunExitsWithAStatusPerOutcome(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{name: "success", args: []string{"firewall", "add", "github.com"}, want: 0},
		{name: "usage error", args: []string{"firewall", "add"}, want: 2},
		{name: "rule broken", args: []string{"firewall", "add", "GitHub.com"}, want: 2},
		{name: "failure", args: []string{"firewall", "rm", "example.com"}, want: 1},
	}
	// Check every outcome in its own home
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aHome(t)

			got, errOut := run(t, tt.args...)

			// Require the status of the outcome
			if got != tt.want {
				t.Errorf("Run(%q) = %d, want %d, stderr:\n%s", tt.args, got, tt.want, errOut)
			}
		})
	}
}

// The file in the home is the one the command changed.
func TestRunWritesTheHomeItRunsIn(t *testing.T) {
	aHome(t)

	status, errOut := run(t, "firewall", "add", "github.com")

	// Stop when the command fails
	if status != 0 {
		t.Fatalf("Run(firewall add github.com) = %d, stderr:\n%s", status, errOut)
	}
	data, err := os.ReadFile(filepath.Join(".", config.File))
	// Require the host in the file
	if err != nil || !strings.Contains(string(data), `"github.com"`) {
		t.Errorf("home file = %q, %v, want github.com in it", data, err)
	}
}
