// Package cli turns a command line into changes to an o3s home.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/secrets"
)

// ErrUsage marks a command line with missing or malformed arguments.
var ErrUsage = errors.New("usage error")

// Streams are where a command reads its input and writes its output.
type Streams struct {
	In       io.Reader
	Out, Err io.Writer
}

// Run executes a command line and hands back its exit status: 0 on success, 2 on a usage error,
// and 1 on any other failure.
func Run(ctx context.Context, args []string, s Streams) int {
	root := newRoot(s)
	root.SetArgs(append([]string{}, args...))
	cmd, err := root.ExecuteContextC(ctx)
	// Report success
	if err == nil {
		return 0
	}
	fmt.Fprintf(s.Err, "Error: %v\n", err)
	// Point a usage error at the help of the command it hit
	if errors.Is(err, ErrUsage) || errors.Is(err, config.ErrInvalid) || errors.Is(err, secrets.ErrInvalid) {
		fmt.Fprintf(s.Err, "Run '%s --help' for usage.\n", cmd.CommandPath())
		return 2
	}
	return 1
}
