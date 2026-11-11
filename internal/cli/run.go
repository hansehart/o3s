// Package cli turns a command line into changes to an o3s home.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/registry"
	"github.com/hansehart/o3s/internal/secrets"
)

// ErrUsage marks a command line with missing or malformed arguments.
var ErrUsage = errors.New("usage error")

// Streams are where a command reads its input and writes its output.
type Streams struct {
	In       io.Reader
	Out, Err io.Writer
}

// Run executes a command line and returns 0, 2 for a usage error, or 1 for any other failure.
func Run(ctx context.Context, args []string, s Streams) int {
	// Lead a user at a terminal through the menu when no command is named
	if len(args) == 0 && isInteractive(s.In, s.Out) {
		return runMenu(ctx, s)
	}
	root := newRoot(s)
	root.SetArgs(append([]string{}, args...))
	cmd, err := root.ExecuteContextC(ctx)
	// Report success
	if err == nil {
		return 0
	}
	fmt.Fprintf(s.Err, "Error: %v\n", err)
	// Point a usage error at the help of the command it hit
	if isUsage(err) {
		fmt.Fprintf(s.Err, "Run '%s --help' for usage.\n", cmd.CommandPath())
		return 2
	}
	return 1
}

// isUsage reports whether an error comes from the command line rather than from the home's files.
func isUsage(err error) bool {
	// Read a broken settings file as a failure of the home, whatever rule it breaks
	if errors.Is(err, config.ErrMalformed) {
		return false
	}
	return errors.Is(err, ErrUsage) || errors.Is(err, config.ErrInvalid) || errors.Is(err, secrets.ErrInvalid) || errors.Is(err, registry.ErrInvalid)
}
