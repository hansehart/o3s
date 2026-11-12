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

// OtherHomes lists the homes this machine records besides one, whose folders still exist.
func OtherHomes(home string) []string {
	records, _ := filepath.Glob(filepath.Join(paths.State(), "*", "home.json"))
	var others []string
	// Read every home's record
	for _, record := range records {
		data, err := os.ReadFile(record)
		var h struct{ Path string }
		// Pass over a record that fails to read
		if err != nil || json.Unmarshal(data, &h) != nil {
			continue
		}
		// Take a home other than this one whose folder still exists
		if _, err := os.Stat(h.Path); err == nil && h.Path != home {
			others = append(others, h.Path)
		}
	}
	return others
}
