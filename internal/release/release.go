// Package release finds, downloads and checks the o3s releases a binary updates itself to.
package release

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/hansehart/o3s/internal/atomicfile"
)

const (
	// repo is where o3s is released.
	repo = "hansehart/o3s"
	// signer is the workflow that signs every release's build attestation.
	signer = repo + "/.github/workflows/publish.yaml"
)

// ErrChecksum marks a download that differs from its line in the release's checksums.
var ErrChecksum = errors.New("differs from the release's checksums")

// base is where the releases live, the environment's choice for a mirror or a test.
func base() string {
	// Take the address the environment names
	if b := os.Getenv("O3S_RELEASES_URL"); b != "" {
		return strings.TrimSuffix(b, "/")
	}
	return "https://github.com/" + repo + "/releases"
}

// Asset is the name a release gives the binary for this host.
func Asset() string {
	return "o3s-linux-" + runtime.GOARCH
}

// Latest reads the newest release's version from where the releases page redirects.
func Latest(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base()+"/latest", nil)
	// Hand back a request that fails to build
	if err != nil {
		return "", err
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	// Hand back releases out of reach
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	tag := path.Base(resp.Header.Get("Location"))
	// Turn away an answer that names no release
	if resp.StatusCode/100 != 3 || !strings.HasPrefix(tag, "v") {
		return "", fmt.Errorf("the releases page names no latest release (%s)", resp.Status)
	}
	return strings.TrimPrefix(tag, "v"), nil
}

// Fetch downloads a release's binary for this host into a folder and checks it against the release's checksums.
func Fetch(ctx context.Context, version, dir string) (string, error) {
	url := base() + "/download/v" + version + "/"
	data, err := get(ctx, url+Asset())
	// Hand back a binary that fails to download
	if err != nil {
		return "", err
	}
	sums, err := get(ctx, url+"checksums.txt")
	// Hand back checksums that fail to download
	if err != nil {
		return "", err
	}
	want, ok := checksum(sums, Asset())
	// Turn away a release whose checksums lack the binary
	if !ok {
		return "", fmt.Errorf("checksums.txt of %s names no %s", version, Asset())
	}
	sum := sha256.Sum256(data)
	// Turn away a binary whose checksum differs
	if hex.EncodeToString(sum[:]) != want {
		return "", fmt.Errorf("%s of %s %w", Asset(), version, ErrChecksum)
	}
	target := filepath.Join(dir, Asset())
	return target, atomicfile.Write(target, data, 0o755)
}

// Verify checks a binary's build attestation where gh is installed and logged in, and reports whether it checked.
func Verify(ctx context.Context, binary string) (bool, error) {
	gh, err := exec.LookPath("gh")
	// Leave the check to the user where gh is missing
	if err != nil {
		return false, nil
	}
	// Leave the check to the user where gh is logged out
	if exec.CommandContext(ctx, gh, "auth", "status").Run() != nil {
		return false, nil
	}
	out, err := exec.CommandContext(ctx, gh, "attestation", "verify", binary, "--repo", repo, "--signer-workflow", signer).CombinedOutput()
	// Turn away a binary whose attestation fails to verify
	if err != nil {
		return false, fmt.Errorf("the build attestation of %s fails to verify: %s", binary, strings.TrimSpace(string(out)))
	}
	return true, nil
}

// VerifyHint is the command a user runs to check a binary's build attestation by hand.
func VerifyHint(binary string) string {
	return "gh attestation verify " + binary + " --repo " + repo + " --signer-workflow " + signer
}

// Newer reports whether version a comes after version b, comparing each part as a number.
func Newer(a, b string) bool {
	x, okA := parts(a)
	y, okB := parts(b)
	// Order nothing that reads as no release
	if !okA || !okB {
		return false
	}
	// Compare the parts from major to patch
	for i := range x {
		// Decide at the first part that differs
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

// parts reads a version's major, minor and patch.
func parts(version string) ([3]int, bool) {
	var out [3]int
	fields := strings.Split(version, ".")
	// Turn away a version without three parts
	if len(fields) != 3 {
		return out, false
	}
	// Read every part as a number
	for i, f := range fields {
		n, err := strconv.Atoi(f)
		// Turn away a part that reads as no number
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// get downloads a URL whole.
func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	// Hand back a request that fails to build
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	// Hand back a download out of reach
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Turn away a download the server lacks
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// checksum finds a file's line in a checksums list.
func checksum(list []byte, name string) (string, bool) {
	sc := bufio.NewScanner(strings.NewReader(string(list)))
	// Look at every line
	for sc.Scan() {
		sum, file, ok := strings.Cut(sc.Text(), "  ")
		// Take the line that names the file
		if ok && file == name {
			return sum, true
		}
	}
	return "", false
}

// IsRelease reports whether a version names a published release.
func IsRelease(version string) bool {
	_, ok := parts(version)
	return ok
}
