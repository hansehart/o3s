package menu

import (
	"testing"

	"charm.land/bubbles/v2/cursor"
	tea "charm.land/bubbletea/v2"
)

// The cursor's blink stays out of the form, so the cursor stays steady, and every other message passes.
func TestSteadyDropsTheCursorBlink(t *testing.T) {
	key := tea.KeyPressMsg{Code: 'a', Text: "a"}

	blink, other := steady(cursor.BlinkMsg{}), steady(key)

	// Require the blink dropped
	if blink != nil {
		t.Errorf("steady(BlinkMsg) = %v, want nil", blink)
	}
	// Require every other message passed on
	if other != key {
		t.Errorf("steady(key) = %v, want the key", other)
	}
}
