package menu

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

// Huh asks each question in a huh form on the terminal, and leaves each answer on screen in one line.
type Huh struct {
	In  io.Reader
	Out io.Writer
}

// Ask shows one question as a form, reading Esc as a step back and Ctrl+C as leaving.
func (h Huh) Ask(q Question) ([]string, error) {
	var single string
	var multi []string
	var ok bool
	// Start with the answer the question offers
	if len(q.Default) > 0 {
		single, multi = q.Default[0], q.Default
	}
	// Show the change the question is about
	for _, line := range q.Preview {
		fmt.Fprintln(h.Out, line)
	}
	interrupted := false
	keys := huh.NewDefaultKeyMap()
	keys.Quit = key.NewBinding(key.WithKeys("esc", "ctrl+c"))
	form := huh.NewForm(huh.NewGroup(field(q, &single, &multi, &ok))).
		WithKeyMap(keys).
		WithShowHelp(false).
		WithProgramOptions(tea.WithInput(h.In), tea.WithOutput(h.Out), tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			// Remember a Ctrl+C, which leaves at once where Esc steps back
			if k, isKey := msg.(tea.KeyPressMsg); isKey && k.String() == "ctrl+c" {
				interrupted = true
			}
			return msg
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
	answer := answerOf(q.Kind, single, multi, ok)
	shown := strings.Join(answer, ", ")
	// Keep a hidden value hidden on screen
	if q.Kind == Hidden {
		shown = strings.Repeat("*", 4)
	}
	label := q.Title
	// Set the answer apart from a title that asks no question
	if !strings.HasSuffix(label, "?") {
		label += ":"
	}
	fmt.Fprintf(h.Out, "> %s %s\n", label, shown)
	return answer, nil
}

// field builds the form field that asks a question, writing its answer into the values given.
func field(q Question, single *string, multi *[]string, ok *bool) huh.Field {
	// Build the field of the question's kind
	switch q.Kind {
	case Multi:
		return huh.NewMultiSelect[string]().Title(q.Title).Options(options(q, *multi)...).Value(multi)
	case Input:
		return huh.NewInput().Title(q.Title).Value(single)
	case Hidden:
		return huh.NewInput().Title(q.Title).EchoMode(huh.EchoModePassword).Value(single)
	case Confirm:
		return huh.NewConfirm().Title(q.Title).Affirmative("Yes").Negative("No").Value(ok)
	case Select:
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
func answerOf(kind Kind, single string, multi []string, ok bool) []string {
	// Read the values of the field's kind
	switch kind {
	case Multi:
		return multi
	case Confirm:
		// Answer yes or no
		if ok {
			return []string{yes}
		}
		return []string{"no"}
	case Select, Input, Hidden:
		return []string{strings.TrimSpace(single)}
	}
	return []string{strings.TrimSpace(single)}
}
