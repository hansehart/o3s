// Package screentest holds the rules every screen of the menu and the output of every command keep, read from their
// text alone, so a test checks them the same way on the menu in memory and on o3s in a real terminal.
package screentest

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// signedURL is a link that carries a query, such as a download link signed for one request.
var signedURL = regexp.MustCompile(`https?://[^\s"]+\?[^\s"]*=`)

// logLine is a line of the devcontainer CLI's own log, which opens with a timestamp in brackets.
var logLine = regexp.MustCompile(`^\s*\[\d{4}-\d\d-\d\dT`)

// footer reports whether a line is the footer: key bindings from the screen's left edge, joined by a middle dot, or
// the binding to go back alone.
func footer(line string) bool {
	return !strings.HasPrefix(line, " ") && (strings.Contains(line, " · ") || strings.HasPrefix(line, "esc "))
}

// Breaks are the rules the lines of a screen of the width and height given break: a line wider than the screen, more
// lines than it holds, a last line other than the footer, a link with a query, and a line of the
// devcontainer CLI's log.
func Breaks(lines []string, width, height int) []string {
	var out []string
	shown := strings.Split(strings.TrimRight(strings.Join(lines, "\n"), "\n "), "\n")
	// Note a screen taller than the terminal
	if len(shown) > height {
		out = append(out, fmt.Sprintf("taller than %d", height))
	}
	// Note a screen whose last line is other than the footer
	if !footer(shown[len(shown)-1]) {
		out = append(out, "no footer")
	}
	// Check every line of the screen
	for _, line := range shown {
		// Note a line wider than the screen
		if ansi.StringWidth(line) > width {
			out = append(out, fmt.Sprintf("wider than %d", width))
		}
		// Note a link with a query
		if signedURL.MatchString(line) {
			out = append(out, "link with a query")
		}
		// Note a line of the devcontainer CLI's log
		if logLine.MatchString(line) {
			out = append(out, "devcontainer log")
		}
	}
	return out
}

// nextStep is a command the user runs next, as o3s names it.
var nextStep = regexp.MustCompile(`\bo3s [a-z]`)

// OutputBreaks are the rules what a command printed breaks: a link with a query, a line of the devcontainer CLI's
// log, and a failure that names no o3s command to run next.
func OutputBreaks(out string, failed bool) []string {
	var breaks []string
	// Note a link with a query
	if signedURL.MatchString(out) {
		breaks = append(breaks, "link with a query")
	}
	// Check every line of the output
	for line := range strings.SplitSeq(out, "\n") {
		// Note a line of the devcontainer CLI's log, once
		if logLine.MatchString(line) {
			breaks = append(breaks, "devcontainer log")
			break
		}
	}
	// Note a failure that names no next step
	if failed && !nextStep.MatchString(out) {
		breaks = append(breaks, "no next step")
	}
	return breaks
}
