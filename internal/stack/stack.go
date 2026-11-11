// Package stack drives the containers an o3s home runs.
package stack

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Restart restarts the home's gateway and then its proxy, which shares the gateway's network, so
// both read the home's settings afresh. It reports whether the home's stack runs.
func Restart(ctx context.Context, home string) (bool, error) {
	project := "label=com.docker.compose.project.working_dir=" + filepath.Join(home, ".devcontainer")
	var ids []string
	// Find the gateway first and the proxy second, the order they restart in
	for _, service := range []string{"gateway", "proxy"} {
		out, err := docker(ctx, "ps", "-q", "--filter", project, "--filter", "label=com.docker.compose.service="+service)
		// Hand back an engine that fails to answer
		if err != nil {
			return false, err
		}
		ids = append(ids, strings.Fields(out)...)
	}
	// Leave a stack that is down to pick the change up when it starts
	if len(ids) == 0 {
		return false, nil
	}
	// Restart every container found, in order
	for _, id := range ids {
		// Hand back a container that fails to restart
		if _, err := docker(ctx, "restart", id); err != nil {
			return true, err
		}
	}
	return true, nil
}

// docker runs the container engine's CLI and hands back what it printed.
func docker(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	// Hand back the engine's own output along with the failure
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker %s: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}
