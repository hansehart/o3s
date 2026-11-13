package menu

import (
	"fmt"
	"io"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// rows is how many options the picker shows at once.
const rows = 10

// picker is a list the user filters by typing and marks with Space or Tab, in one go.
type picker struct {
	q      Question
	filter string
	cursor int
	marked map[string]bool
	answer []string
	err    error
}

// newPicker starts a picker on a question, its defaults marked.
func newPicker(q Question) picker {
	marked := map[string]bool{}
	// Mark every default
	for _, d := range q.Default {
		marked[d] = true
	}
	return picker{q: q, marked: marked}
}

// visible lists the options that contain the filter, in the question's order.
func (p picker) visible() []Option {
	var out []Option
	// Keep every option that matches
	for _, o := range p.q.Options {
		// Take an option that contains the filter
		if strings.Contains(strings.ToLower(o.Value), strings.ToLower(p.filter)) {
			out = append(out, o)
		}
	}
	return out
}

// Init starts the picker without a command.
func (picker) Init() tea.Cmd {
	return nil
}

// Update follows a key: typing filters, Space and Tab mark, Enter hands back, Esc clears or steps back.
func (p picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, isKey := msg.(tea.KeyPressMsg)
	// Keep the list as it is on anything but a key
	if !isKey {
		return p, nil
	}
	shown := p.visible()
	// Follow the key the user pressed
	switch k.String() {
	case "ctrl+c":
		p.err = ErrQuit
		return p, tea.Quit
	case "esc":
		// Step back where nothing is filtered, else clear the filter
		if p.filter == "" {
			p.err = ErrBack
			return p, tea.Quit
		}
		p.filter, p.cursor = "", 0
	case "enter":
		p.answer = p.chosen(shown)
		return p, tea.Quit
	case "space", "tab":
		// Mark or unmark the highlighted option
		if p.cursor < len(shown) {
			v := shown[p.cursor].Value
			p.marked[v] = !p.marked[v]
		}
	case "up", "shift+tab":
		p.cursor = max(p.cursor-1, 0)
	case "down":
		p.cursor = min(p.cursor+1, max(len(shown)-1, 0))
	case "backspace":
		// Drop the last character of the filter
		if r := []rune(p.filter); len(r) > 0 {
			p.filter, p.cursor = string(r[:len(r)-1]), 0
		}
	default:
		// Take a typed character into the filter
		if k.Text != "" {
			p.filter, p.cursor = p.filter+k.Text, 0
		}
	}
	return p, nil
}

// chosen hands back the marked options in the question's order, or the highlighted one where none is marked.
func (p picker) chosen(shown []Option) []string {
	var out []string
	// Take every marked option in order
	for _, o := range p.q.Options {
		// Take an option the user marked
		if p.marked[o.Value] {
			out = append(out, o.Value)
		}
	}
	// Take the highlighted option where the user marked none
	if len(out) == 0 && p.cursor < len(shown) {
		out = []string{shown[p.cursor].Value}
	}
	return out
}

// View draws the title, the filter, the options in reach of the cursor, and the keys.
func (p picker) View() tea.View {
	var b strings.Builder
	fmt.Fprintf(&b, "> %s\n  / %s\n", p.q.Title, p.filter)
	shown := p.visible()
	width := 0
	// Measure the longest option, so the notes line up
	for _, o := range shown {
		width = max(width, len(o.Value))
	}
	start := max(p.cursor-rows+1, 0)
	// Draw the options in reach of the cursor
	for i := start; i < min(start+rows, len(shown)); i++ {
		o := shown[i]
		pointer, box := "  ", "[ ]"
		// Point at the highlighted option
		if i == p.cursor {
			pointer = "▸ "
		}
		// Fill the box of a marked option
		if p.marked[o.Value] {
			box = "[•]"
		}
		fmt.Fprintf(&b, "%s%s %-*s   %s\n", pointer, box, width, o.Value, o.Note)
	}
	// Name a filter that matches nothing
	if len(shown) == 0 {
		b.WriteString("  no match\n")
	}
	b.WriteString("\ntype to filter · space mark · enter done · esc back")
	return tea.NewView(b.String())
}

// Pick asks a multi-select question in a picker and hands back the options chosen.
func Pick(q Question, in io.Reader, out io.Writer) ([]string, error) {
	m, err := tea.NewProgram(newPicker(q), tea.WithInput(in), tea.WithOutput(out), tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg { return steady(msg) })).Run()
	// Hand back a picker that fails to run
	if err != nil {
		return nil, err
	}
	p := m.(picker)
	return slices.Clip(p.answer), p.err
}
