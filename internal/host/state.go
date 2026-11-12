package host

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/hansehart/o3s/internal/atomicfile"
	"github.com/hansehart/o3s/internal/paths"
)

// HomeState makes the folder o3s keeps for a home, with the record that names it and the folder the nested daemon shares its socket in.
func HomeState(home string) error {
	dir := paths.Home(home)
	// Make each folder for the user alone, as the nested daemon keeps its socket folder
	for _, d := range []string{dir, filepath.Join(dir, "docker-host")} {
		// Hand back a folder that stays uncreated
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	record, err := json.MarshalIndent(struct {
		Path string `json:"path"`
	}{home}, "", "  ")
	// Hand back a record that fails to encode
	if err != nil {
		return err
	}
	return atomicfile.Write(filepath.Join(dir, "home.json"), append(record, '\n'), 0o600)
}
