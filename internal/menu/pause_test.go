package menu

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Enter and Esc go back to the menu, Ctrl+C leaves o3s, and any other key keeps waiting.
func TestPauseWaitsForEnterOrEsc(t *testing.T) {
	tests := []struct {
		key  tea.KeyPressMsg
		done bool
		want error
	}{
		{tea.KeyPressMsg{Code: tea.KeyEnter}, true, nil},
		{tea.KeyPressMsg{Code: tea.KeyEscape}, true, nil},
		{tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, true, ErrQuit},
		{tea.KeyPressMsg{Code: 'a', Text: "a"}, false, nil},
	}
	// Press every key on a fresh pause
	for _, tt := range tests {
		m, cmd := pause{}.Update(tt.key)

		got := m.(pause)
		// Report a key that ends the wait otherwise
		if (cmd != nil) != tt.done || !errors.Is(got.err, tt.want) {
			t.Errorf("Update(%s) = done %v, %v, want done %v, %v", tt.key, cmd != nil, got.err, tt.done, tt.want)
		}
	}
}
