package menu

import (
	"slices"
	"strings"
)

// collapse writes the line an answered question leaves on screen, the way the dialogs show it.
func collapse(q Question, answer []string) string {
	label := "> " + q.Title
	// Answer a confirmation with its letter after the choices it offered
	if q.Kind == Confirm {
		letter := "n"
		// Show yes as y
		if len(answer) > 0 && answer[0] == yes {
			letter = "y"
		}
		return label + " [y/N] " + letter
	}
	shown := strings.Join(answer, ", ")
	// Keep a hidden value hidden on screen
	if q.Kind == Hidden {
		shown = "****"
	}
	// Show the default in brackets, and the answer where it differs from it
	if len(q.Default) > 0 {
		label += " [" + strings.Join(q.Default, ",") + "]:"
		// Leave the line at the default where the user kept it
		if slices.Equal(answer, q.Default) {
			return label
		}
		return label + " " + shown
	}
	// Set the answer apart from a title that asks no question
	if !strings.HasSuffix(q.Title, "?") {
		label += ":"
	}
	return label + " " + shown
}
