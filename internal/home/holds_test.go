package home_test

import (
	"testing"

	"github.com/hansehart/o3s/internal/home"
)

// A home that would hold one of the host's own folders is named, and a folder beside them is fine.
func TestHoldsNamesAFolderInsideTheHome(t *testing.T) {
	host := []string{"/home/ubuntu/.config/o3s", "/home/ubuntu/.local/state/o3s"}
	tests := []struct {
		dir  string
		want string
	}{
		{"/home/ubuntu", "/home/ubuntu/.config/o3s"},
		{"/home/ubuntu/.config", "/home/ubuntu/.config/o3s"},
		{"/home/ubuntu/.local/state/o3s", "/home/ubuntu/.local/state/o3s"},
		{"/home/ubuntu/o3s", ""},
		{"/home/ubuntu/.config/o3sx", ""},
	}
	// Check every home
	for _, tt := range tests {
		got, ok := home.Holds(tt.dir, host...)

		// Report a home whose verdict differs
		if got != tt.want || ok != (tt.want != "") {
			t.Errorf("Holds(%s) = %q, %v, want %q", tt.dir, got, ok, tt.want)
		}
	}
}
