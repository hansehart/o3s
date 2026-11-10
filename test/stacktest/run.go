package stacktest

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// run runs a command on the host, and hands back what it printed.
func run(ctx context.Context, tb testing.TB, name string, args ...string) (string, error) {
	tb.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	// Hand back the command's own output along with the failure
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("%s %s: %w\n%s%s", name, strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}
