package screentest_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/fixture"
	"github.com/hansehart/o3s/internal/tui/screentest"
)

// footer is the last line of a screen that keeps the rules.
const footer = "↑/↓ navigate · esc back"

// A screen breaks exactly the rules its text breaks, each named once per line that breaks it.
func TestBreaks(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  []string
	}{
		{"a screen that fits", []string{fixture.Host, "", footer, ""}, nil},
		{"a line wider than the screen", []string{strings.Repeat("x", 41), footer}, []string{"wider than 40"}},
		{"more lines than the screen holds", []string{"1", "2", "3", "4", footer}, []string{"taller than 4"}},
		{"a footer cut off", []string{fixture.Host, fixture.OtherHost}, []string{"no footer"}},
		{"a footer cut off below a row", []string{fixture.Host, "  cage · stopped"}, []string{"no footer"}},
		{"a footer of the binding to go back alone", []string{fixture.Host, "esc back"}, nil},
		{"a footer cut to the width", []string{fixture.Host, "↑/↓ navigate · enter select …"}, nil},
		{"a link with a query", []string{"get https://" + fixture.Host + "/blob?hmac=" + fixture.SecretValue, footer}, []string{"link with a query"}},
		{"a line of the devcontainer log", []string{"[" + fixture.NowText + "] Start: Run", footer}, []string{"devcontainer log"}},
	}
	// Check every screen
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := screentest.Breaks(tt.lines, 40, 4)

			// Report other breaks than the screen's
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Breaks() (-want +got):\n%s", diff)
			}
		})
	}
}

// A command's output breaks exactly the rules its text breaks, a failure's next step included.
func TestOutputBreaks(t *testing.T) {
	tests := []struct {
		name   string
		out    string
		failed bool
		want   []string
	}{
		{"a success", "Added " + fixture.Repo + "\n", false, nil},
		{"a failure that names the next step", "error: the firewall blocks " + fixture.Host + ", allow it with: o3s firewall add " + fixture.Host + "\n", true, nil},
		{"a failure that names none", "error: exit status 1\n", true, []string{"no next step"}},
		{"a link with a query", "error: get https://" + fixture.Host + "/blob?hmac=" + fixture.SecretValue + ", run o3s update\n", true, []string{"link with a query"}},
		{"lines of the devcontainer log", "[" + fixture.NowText + "] Start: Run\n[" + fixture.NowText + "] Stop\nAdded " + fixture.Repo + "\n", false, []string{"devcontainer log"}},
	}
	// Check every output
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := screentest.OutputBreaks(tt.out, tt.failed)

			// Report other breaks than the output's
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("OutputBreaks() (-want +got):\n%s", diff)
			}
		})
	}
}
