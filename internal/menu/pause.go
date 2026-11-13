package menu

import (
	"io"

	tea "charm.land/bubbletea/v2"
)

// pause waits on the output of an action until the user goes back to the menu or leaves.
type pause struct {
	err error
}

// Init starts the wait without a command.
func (pause) Init() tea.Cmd {
	return nil
}

// Update ends the wait on Enter or Esc, and leaves o3s on Ctrl+C.
func (p pause) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, isKey := msg.(tea.KeyPressMsg)
	// Keep waiting on anything but a key
	if !isKey {
		return p, nil
	}
	// Follow the key the user pressed
	switch k.String() {
	case "enter", "esc":
		return p, tea.Quit
	case "ctrl+c":
		p.err = ErrQuit
		return p, tea.Quit
	}
	return p, nil
}

// View shows how to go back.
func (pause) View() tea.View {
	return tea.NewView("\n↵/esc back to the menu")
}

// Pause shows the hint under an action's output and waits until the user goes back, or reports that they left.
func Pause(in io.Reader, out io.Writer) error {
	m, err := tea.NewProgram(pause{}, tea.WithInput(in), tea.WithOutput(out), tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg { return steady(msg) })).Run()
	// Hand back a wait that fails to run
	if err != nil {
		return err
	}
	return m.(pause).err
}
