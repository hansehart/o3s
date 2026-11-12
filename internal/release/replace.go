package release

import (
	"os"

	"github.com/hansehart/o3s/internal/atomicfile"
)

// Replace puts a new binary at a binary's path in a single step, so the running process keeps its own.
func Replace(path, src string) error {
	data, err := os.ReadFile(src)
	// Hand back a new binary that fails to read
	if err != nil {
		return err
	}
	return atomicfile.Write(path, data, 0o755)
}
