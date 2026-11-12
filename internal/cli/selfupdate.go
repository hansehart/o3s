package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/hansehart/o3s/internal/release"
)

var (
	// executable finds the running binary's own file, which an update replaces.
	executable = func() (string, error) {
		path, err := os.Executable()
		// Hand back a binary whose file stays unknown
		if err != nil {
			return "", err
		}
		return filepath.EvalSymlinks(path)
	}
	// restartAs turns the running process into another binary, which carries the update on.
	restartAs = func(path string, args []string) error {
		return syscall.Exec(path, args, os.Environ())
	}
)

// selfUpdate brings the binary to the latest release and hands the rest of the update to it, and reports whether it did.
func selfUpdate(ctx context.Context, in io.Reader, out io.Writer, yes bool) (bool, error) {
	current := version()
	// Leave a dev build to the way it was built
	if !release.IsRelease(current) {
		fmt.Fprintln(out, "o3s dev builds update with go install or install.sh")
		return false, nil
	}
	latest, err := release.Latest(ctx)
	// Go on with this binary where the releases stay out of reach
	if err != nil {
		fmt.Fprintf(out, "⚠ releases out of reach, o3s stays at %s\n", current)
		return false, nil
	}
	// Go on with this binary where it is the latest release
	if !release.Newer(latest, current) {
		return false, nil
	}
	fmt.Fprintf(out, "%-14s %s → %s\n", "o3s", current, latest)
	ok, err := confirm(in, out, isTerminal(in), yes)
	// Hand back an answer that fails to arrive, or go on with this binary when the user declines
	if err != nil || !ok {
		return false, err
	}
	path, err := executable()
	// Hand back a binary whose file stays unknown
	if err != nil {
		return false, err
	}
	dir, err := os.MkdirTemp("", "o3s-update-")
	// Hand back a download that finds no folder
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(dir)
	bin, err := release.Fetch(ctx, latest, dir)
	// Hand back a release that fails to download or to match its checksum
	if err != nil {
		return false, err
	}
	verified, err := release.Verify(ctx, bin)
	// Hand back a release whose attestation fails to verify
	if err != nil {
		return false, err
	}
	// Hand back a binary that stays in place
	if err := release.Replace(path, bin); err != nil {
		return false, err
	}
	fmt.Fprintf(out, "Updated o3s to %s\n", latest)
	// Show how to check the attestation where gh checked none
	if !verified {
		fmt.Fprintf(out, "Verify its build attestation with: %s\n", release.VerifyHint(path))
	}
	args := []string{path, "update"}
	// Pass the answer on to the new binary's own question
	if yes {
		args = append(args, "--yes")
	}
	_ = os.RemoveAll(dir)
	return true, restartAs(path, args)
}
