package usagetest

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/hansehart/o3s/internal/menu"
	"github.com/hansehart/o3s/internal/ptytest"
)

// Answer gives the question on the screen the answer a path of the menu holds, the way a user does, reading only what
// the terminal shows: the arrows to the row and Enter, the marks with Space, y or n, the line typed, the tab with the
// arrows and its key.
func Answer(s *ptytest.Session, q menu.Question, values []string) error {
	var err error
	// Answer by the kind of question
	switch q.Kind {
	case menu.Select:
		err = choose(s, q.Options, values[0])
	case menu.Multi:
		err = markOnly(s, q.Options, values)
		err = then(err, s, "enter")
	case menu.Confirm:
		key := "n"
		// Press y for yes
		if values[0] == "yes" {
			key = "y"
		}
		err = s.Press(key)
	case menu.Input, menu.Hidden:
		err = s.Press("ctrl+u", values[0], "enter")
	case menu.View:
		err = inView(s, q.Tabs, values)
	default:
		err = fmt.Errorf("no way to answer a question of kind %d", q.Kind)
	}
	s.Settle()
	return err
}

// then presses the keys where the step before went well.
func then(err error, s *ptytest.Session, keys ...string) error {
	// Keep the failure of the step before
	if err != nil {
		return err
	}
	return s.Press(keys...)
}

// choose moves to the row of the value and presses Enter.
func choose(s *ptytest.Session, options []menu.Option, value string) error {
	return then(moveTo(s, options, value), s, "enter")
}

// row matches the line of the highlighted row that shows the option: the pointer, a number or a mark where the list
// shows them, the label, and the columns after it.
func row(o menu.Option) *regexp.Regexp {
	var pattern strings.Builder
	pattern.WriteString(`(?m)^[❯↕] (\d+\. )?(\[.\] )?` + regexp.QuoteMeta(o.Shown()))
	// Follow the label with every column
	for _, c := range o.Columns {
		pattern.WriteString(`\s+` + regexp.QuoteMeta(c))
	}
	return regexp.MustCompile(pattern.String() + `(\s|$)`)
}

// moveTo presses ↓, then ↑, until the row of the value is highlighted.
func moveTo(s *ptytest.Session, options []menu.Option, value string) error {
	i := slices.IndexFunc(options, func(o menu.Option) bool { return o.Value == value })
	// Turn away a value no option holds
	if i < 0 {
		return fmt.Errorf("no option %q", value)
	}
	re := row(options[i])
	// Look down the list, then up it
	for _, key := range []string{"down", "up"} {
		// Press until the row is highlighted or the list is passed
		for range len(options) + 1 {
			s.Settle()
			// Stop at the row
			if re.MatchString(s.Screen()) {
				return nil
			}
			// Hand back a terminal that takes no more keys
			if err := s.Press(key); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("no row %q on:\n%s", options[i].Shown(), s.Screen())
}

// marked reports whether the highlighted row holds a mark.
func marked(s *ptytest.Session) bool {
	return regexp.MustCompile(`(?m)^[❯↕] (\d+\. )?\[x\]`).MatchString(s.Screen())
}

// markOnly turns the marks of the list until exactly the values given are marked.
func markOnly(s *ptytest.Session, options []menu.Option, values []string) error {
	// Turn every row whose mark differs
	for _, o := range options {
		// Hand back a row out of reach
		if err := moveTo(s, options, o.Value); err != nil {
			return err
		}
		// Turn a row whose mark differs from the one wanted
		if marked(s) != slices.Contains(values, o.Value) {
			// Hand back a terminal that takes no more keys
			if err := s.Press("space"); err != nil {
				return err
			}
		}
	}
	return nil
}

// inView switches to the tab the answer names, then answers on it: its key, a new order, its marks, or a row.
func inView(s *ptytest.Session, tabs []menu.Tab, values []string) error {
	i := slices.IndexFunc(tabs, func(t menu.Tab) bool { return t.Name == values[0] })
	// Turn away a tab the view lacks
	if i < 0 {
		return fmt.Errorf("no tab %q", values[0])
	}
	// Hand back a tab out of reach
	if err := toTab(s, tabs, tabs[i].Name); err != nil {
		return err
	}
	tab, rest := tabs[i], values[1:]
	k := slices.IndexFunc(tab.Keys, func(k menu.Key) bool { return len(rest) > 0 && k.Value == rest[0] })
	// Answer the way the tab takes it
	switch {
	case k >= 0:
		return s.Press(tab.Keys[k].Key)
	case len(rest) > 1 && rest[0] == menu.Order:
		return then(moveTo(s, tab.Options, rest[1]), s, "m", "down", "enter")
	case tab.Multi:
		return then(markOnly(s, tab.Options, rest), s, "enter")
	default:
		return choose(s, tab.Options, rest[0])
	}
}

// toTab presses → until the tab named is the one lit, where the view shows its tabs.
func toTab(s *ptytest.Session, tabs []menu.Tab, name string) error {
	// Take the one tab of a view that shows no tabs
	if len(tabs) == 1 {
		return nil
	}
	// Press until the tab is lit or every tab was
	for range len(tabs) + 1 {
		s.Settle()
		// Stop at the tab
		if slices.ContainsFunc(s.Lit(), func(lit string) bool { return strings.TrimSpace(lit) == name }) {
			return nil
		}
		// Hand back a terminal that takes no more keys
		if err := s.Press("right"); err != nil {
			return err
		}
	}
	return fmt.Errorf("no tab %q lit on:\n%s", name, s.Screen())
}
