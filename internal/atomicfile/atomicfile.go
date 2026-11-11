// Package atomicfile replaces files in a single step, so a reader always finds one whole.
package atomicfile

import (
	"errors"
	"os"
	"path/filepath"
)

// Write replaces a file with the data given, open to those the mode names.
func Write(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	// Hand back a folder that takes no new file
	if err != nil {
		return err
	}
	// Clean up the temporary file whenever it stays behind
	defer os.Remove(tmp.Name())
	// Open the file to those the mode names
	if err := tmp.Chmod(perm); err != nil {
		return errors.Join(err, tmp.Close())
	}
	// Hand back a write that fails part way
	if _, err := tmp.Write(data); err != nil {
		return errors.Join(err, tmp.Close())
	}
	// Hand back a file that fails to close
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
