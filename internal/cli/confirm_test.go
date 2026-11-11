package cli

import (
	"errors"
	"strings"
	"testing"
)

// The user's answer at a terminal decides, and --yes applies without asking.
func TestConfirmFollowsTheAnswer(t *testing.T) {
	tests := []struct {
		answer   string
		terminal bool
		yes      bool
		want     bool
	}{
		{"y\n", true, false, true},
		{"yes\n", true, false, true},
		{"n\n", true, false, false},
		{"\n", true, false, false},
		{"", true, false, false},
		{"", false, true, true},
	}
	// Check every answer
	for _, tt := range tests {
		var out strings.Builder

		got, err := confirm(strings.NewReader(tt.answer), &out, tt.terminal, tt.yes)

		// Report an answer read otherwise
		if err != nil || got != tt.want {
			t.Errorf("confirm(%q, terminal %v, yes %v) = %v, %v, want %v", tt.answer, tt.terminal, tt.yes, got, err, tt.want)
		}
	}
}

// Without a terminal and without --yes, nothing can answer, so the command is a usage error.
func TestConfirmTurnsAwayAMissingTerminal(t *testing.T) {
	var out strings.Builder

	_, err := confirm(strings.NewReader("y\n"), &out, false, false)

	// Require a usage error
	if !errors.Is(err, ErrUsage) {
		t.Errorf("confirm() without a terminal = %v, want ErrUsage", err)
	}
}
