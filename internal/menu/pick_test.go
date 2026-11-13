package menu

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/google/go-cmp/cmp"
)

// features is a store's list the way the feature question offers it.
var features = Question{Key: "feature", Kind: Multi, Title: "Features", Options: []Option{{Value: "claude-code"}, {Value: "codex"}, {Value: "go"}, {Value: "golangci-lint"}, {Value: "rust"}}}

// press sends keys to a picker in turn and hands back where it ends, and whether the last key ended it.
func press(p picker, keys ...tea.KeyPressMsg) (picker, bool) {
	var cmd tea.Cmd
	// Send every key in turn
	for _, k := range keys {
		var m tea.Model
		m, cmd = p.Update(k)
		p = m.(picker)
	}
	return p, cmd != nil
}

// typed is the key press of a printable character.
func typed(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

var (
	space = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	tab   = tea.KeyPressMsg{Code: tea.KeyTab}
	enter = tea.KeyPressMsg{Code: tea.KeyEnter}
	esc   = tea.KeyPressMsg{Code: tea.KeyEscape}
	down  = tea.KeyPressMsg{Code: tea.KeyDown}
	ctrlC = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
)

// Typing narrows the list to the options that contain what was typed.
func TestPickFiltersAsYouType(t *testing.T) {
	p, _ := press(newPicker(features), typed('g'), typed('o'))

	got := values(p.visible())

	// Report a list narrowed otherwise
	if diff := cmp.Diff([]string{"go", "golangci-lint"}, got); diff != "" {
		t.Errorf("visible (-want +got):\n%s", diff)
	}
}

// Space and Tab mark the highlighted option and keep the filter, and Enter hands back the marks in the store's order.
func TestPickMarksWhileFiltering(t *testing.T) {
	p, done := press(newPicker(features), typed('r'), typed('u'), space, esc, typed('c'), typed('l'), tab, enter)

	// Require the marks in the store's order
	if diff := cmp.Diff([]string{"claude-code", "rust"}, p.answer); diff != "" || !done {
		t.Errorf("answer (-want +got):\n%s, done %v", diff, done)
	}
}

// A mark keeps the filter, so the next option of the same search is one key away.
func TestPickKeepsTheFilterAfterAMark(t *testing.T) {
	p, _ := press(newPicker(features), typed('g'), typed('o'), space)

	// Require the filter kept
	if p.filter != "go" {
		t.Errorf("filter = %q, want go", p.filter)
	}
}

// Enter without a mark takes the highlighted option.
func TestPickTakesTheHighlightedOptionWithoutAMark(t *testing.T) {
	p, done := press(newPicker(features), down, down, enter)

	// Require the highlighted option
	if diff := cmp.Diff([]string{"go"}, p.answer); diff != "" || !done {
		t.Errorf("answer (-want +got):\n%s, done %v", diff, done)
	}
}

// Esc clears the filter first and steps back on an empty one, and Ctrl+C leaves at once.
func TestPickEscClearsThenStepsBack(t *testing.T) {
	cleared, clearedDone := press(newPicker(features), typed('g'), esc)
	back, backDone := press(newPicker(features), esc)
	quit, quitDone := press(newPicker(features), typed('g'), ctrlC)

	// Require the filter cleared without ending
	if cleared.filter != "" || clearedDone {
		t.Errorf("esc on a filter = %q, done %v, want an empty filter, open", cleared.filter, clearedDone)
	}
	// Require a step back on an empty filter, and leaving on Ctrl+C
	if !errors.Is(back.err, ErrBack) || !backDone || !errors.Is(quit.err, ErrQuit) || !quitDone {
		t.Errorf("esc = %v, %v and ctrl+c = %v, %v, want ErrBack and ErrQuit", back.err, backDone, quit.err, quitDone)
	}
}

// The question's defaults start marked, so a plain Enter keeps them.
func TestPickStartsWithTheDefaultsMarked(t *testing.T) {
	q := Question{Key: "inject", Kind: Multi, Options: []Option{{Value: "header"}, {Value: "query"}, {Value: "body"}}, Default: []string{"header"}}

	p, _ := press(newPicker(q), enter)

	// Require the default handed back
	if diff := cmp.Diff([]string{"header"}, p.answer); diff != "" {
		t.Errorf("answer (-want +got):\n%s", diff)
	}
}

// values lists the values of options.
func values(options []Option) []string {
	var out []string
	// Take every value in order
	for _, o := range options {
		out = append(out, o.Value)
	}
	return out
}
