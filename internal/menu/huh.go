package menu

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

// Huh asks each question in a huh form on the terminal, and leaves each answer on screen in one line.
type Huh struct {
	In  io.Reader
	Out io.Writer
}

// Ask shows one question as a form, or a multi-select in the picker, reading Esc as a step back and Ctrl+C as leaving.
func (h Huh) Ask(q Question) ([]string, error) {
	var single string
	var ok bool
	// Start with the answer the question offers
	if len(q.Default) > 0 {
		single = q.Default[0]
	}
	// Show the change the question is about
	for _, line := range q.Preview {
		fmt.Fprintln(h.Out, line)
	}
	// Ask a multi-select in the picker, which filters and marks in one go
	if q.Kind == Multi {
		answer, err := Pick(q, h.In, h.Out)
		// Hand back a step back, a leave, or a picker that fails
		if err != nil {
			return nil, err
		}
		fmt.Fprintln(h.Out, collapse(q, answer))
		return answer, nil
	}
	interrupted := false
	keys := huh.NewDefaultKeyMap()
	keys.Quit = key.NewBinding(key.WithKeys("esc", "ctrl+c"))
	form := huh.NewForm(huh.NewGroup(field(q, &single, &ok))).
		WithKeyMap(keys).
		WithShowHelp(false).
		WithProgramOptions(tea.WithInput(h.In), tea.WithOutput(h.Out), tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			// Remember a Ctrl+C, which leaves at once where Esc steps back
			if k, isKey := msg.(tea.KeyPressMsg); isKey && k.String() == "ctrl+c" {
				interrupted = true
			}
			return steady(msg)
		}))
	err := form.Run()
	// Tell leaving from stepping back
	if errors.Is(err, huh.ErrUserAborted) {
		// Leave at once on Ctrl+C
		if interrupted {
			return nil, ErrQuit
		}
		return nil, ErrBack
	}
	// Hand back a form that fails to run
	if err != nil {
		return nil, err
	}
	answer := answerOf(q.Kind, single, ok)
	// Take the default where the user left an input with one empty
	if q.Kind == Input && len(q.Default) > 0 && answer[0] == "" {
		answer = q.Default
	}
	fmt.Fprintln(h.Out, collapse(q, answer))
	return answer, nil
}

// field builds the form field that asks a question, writing its answer into the values given.
func field(q Question, single *string, ok *bool) huh.Field {
	// Build the field of the question's kind
	switch q.Kind {
	case Input:
		// Ask with the default in brackets, the field empty so a plain Enter keeps it
		if len(q.Default) > 0 {
			*single = ""
			return huh.NewInput().Title(q.Title + " [" + q.Default[0] + "]").Placeholder(q.Default[0]).Value(single)
		}
		return huh.NewInput().Title(q.Title).Value(single)
	case Hidden:
		return huh.NewInput().Title(q.Title).EchoMode(huh.EchoModePassword).Value(single)
	case Confirm:
		return huh.NewConfirm().Title(q.Title).Affirmative("Yes").Negative("No").Value(ok)
	case Select, Multi:
		return huh.NewSelect[string]().Title(q.Title).Options(options(q, nil)...).Filtering(true).Value(single)
	}
	return huh.NewInput().Title(q.Title).Value(single)
}

// options lists a question's choices with their notes beside them, the chosen ones marked.
func options(q Question, chosen []string) []huh.Option[string] {
	width := 0
	// Measure the longest choice, so the notes line up
	for _, o := range q.Options {
		width = max(width, len(o.Value))
	}
	out := make([]huh.Option[string], len(q.Options))
	// Build every choice
	for i, o := range q.Options {
		label := o.Value
		// Put the note beside its choice
		if o.Note != "" {
			label = fmt.Sprintf("%-*s   %s", width, o.Value, o.Note)
		}
		out[i] = huh.NewOption(label, o.Value).Selected(slices.Contains(chosen, o.Value))
	}
	return out
}

// answerOf hands back the values a field of a kind collected.
func answerOf(kind Kind, single string, ok bool) []string {
	// Answer a confirmation with yes or no
	if kind == Confirm && ok {
		return []string{yes}
	}
	// Answer a declined confirmation with no
	if kind == Confirm {
		return []string{"no"}
	}
	return []string{strings.TrimSpace(single)}
}

// steady keeps the cursor's blink out of the form, so an input's cursor stays shown without flashing.
func steady(msg tea.Msg) tea.Msg {
	// Drop the message that would hide the cursor half the time
	if _, blink := msg.(cursor.BlinkMsg); blink {
		return nil
	}
	return msg
}
