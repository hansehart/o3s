// Package ptytest runs a program in a pseudo-terminal and reads its screen back through a terminal emulator, the way a
// user's terminal shows it, for tests that drive o3s with keys.
package ptytest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
)

// Keys are the keys a test names, as a terminal sends them.
var Keys = map[string]string{
	"enter":  "\r",
	"esc":    "\x1b",
	"up":     "\x1b[A",
	"down":   "\x1b[B",
	"right":  "\x1b[C",
	"left":   "\x1b[D",
	"tab":    "\t",
	"space":  " ",
	"ctrl+c": "\x03",
	"ctrl+u": "\x15",
}

// Wait bounds how long a test waits for a screen or for the program to leave, three times as long on CI's slower
// machines.
func Wait() time.Duration {
	// Wait longer on CI
	if os.Getenv("CI") != "" {
		return 15 * time.Second
	}
	return 5 * time.Second
}

// Session is a program running in a pseudo-terminal, read back through a terminal emulator.
type Session struct {
	cmd  *exec.Cmd
	pty  *os.File
	term *vt.Emulator
	done chan struct{}
	err  error
	mu   sync.Mutex
}

// Start runs the command in a pseudo-terminal of the size given, in the folder and with the environment given, with a
// terminal that takes colors.
func Start(dir string, env []string, cols, rows int, name string, args ...string) (*Session, error) {
	cmd := exec.CommandContext(context.Background(), name, args...)
	cmd.Dir = dir
	cmd.Env = append(slices.Clone(env), "TERM=xterm-256color")
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)}) //nolint:gosec // the sizes come from the test
	// Hand back a program that fails to start
	if err != nil {
		return nil, err
	}
	s := &Session{cmd: cmd, pty: f, term: vt.NewEmulator(cols, rows), done: make(chan struct{})}
	go func() { _, _ = io.Copy(s, f) }()
	go func() { _, _ = io.Copy(f, s.term) }()
	go func() {
		err := cmd.Wait()
		s.mu.Lock()
		s.err = err
		s.mu.Unlock()
		close(s.done)
	}()
	return s, nil
}

// Write hands what the program prints to the terminal emulator, one write at a time, so the screen is read whole.
func (s *Session) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.term.Write(p)
}

// Screen is the text the terminal emulator shows now.
func (s *Session) Screen() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.term.String()
}

// Press presses keys one after another: a key by its name in Keys, any other word typed as it reads, with a moment
// between them as a user's hands leave.
func (s *Session) Press(keys ...string) error {
	// Press every key in turn
	for _, k := range keys {
		seq, named := Keys[k]
		// Type a word that names no key
		if !named {
			seq = k
		}
		// Hand back a terminal that takes no more keys
		if _, err := io.WriteString(s.pty, seq); err != nil {
			return err
		}
		pause := 50 * time.Millisecond
		// Give a lone Esc the time a terminal leaves before it reads it as a key of its own, not the start of a sequence
		if seq == Keys["esc"] {
			pause = 300 * time.Millisecond
		}
		time.Sleep(pause)
	}
	return nil
}

// Resize resizes the terminal, the way a user's window does.
func (s *Session) Resize(cols, rows int) error {
	s.mu.Lock()
	s.term.Resize(cols, rows)
	s.mu.Unlock()
	return pty.Setsize(s.pty, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)}) //nolint:gosec // the sizes come from the test
}

// Until checks a condition on the screen again and again until it holds, and reports whether it did within Wait.
func (s *Session) Until(holds func(screen string) bool) bool {
	return Within(func() bool { return holds(s.Screen()) })
}

// Left reports whether the program left within Wait, and the error it left with.
func (s *Session) Left() (bool, error) {
	left := Within(func() bool {
		// Report a program that left
		select {
		case <-s.done:
			return true
		default:
			return false
		}
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	return left, s.err
}

// Stop ends a program still running and waits for it to leave.
func (s *Session) Stop() {
	// Kill a program still running
	select {
	case <-s.done:
	default:
		_ = s.cmd.Process.Kill()
		<-s.done
	}
}

// Failed reports whether the error a program left with is its own failure, not a failure to run it.
func Failed(err error) bool {
	var exit *exec.ExitError
	return errors.As(err, &exit)
}

// Within checks a condition again and again until it holds, and reports whether it did within Wait.
func Within(holds func() bool) bool {
	end := time.Now().Add(Wait())
	// Check until the condition holds or the wait is over
	for time.Now().Before(end) {
		// Report a condition that holds
		if holds() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return holds()
}

// Size reads a size written as COLSxROWS.
func Size(text string) (int, int, error) {
	c, r, ok := strings.Cut(text, "x")
	cols, cerr := strconv.Atoi(c)
	rows, rerr := strconv.Atoi(r)
	// Turn away a size that reads as none
	if !ok || cerr != nil || rerr != nil || cols < 1 || rows < 1 {
		return 0, 0, fmt.Errorf("size %q, want COLSxROWS", text)
	}
	return cols, rows, nil
}

// Lit is the text the screen shows on a background of its own, such as the tab in front, one run of cells per line.
func (s *Session) Lit() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	b := s.term.Bounds()
	// Read every line of the screen
	for y := b.Min.Y; y < b.Max.Y; y++ {
		var run strings.Builder
		// Take every cell on a background of its own
		for x := b.Min.X; x < b.Max.X; x++ {
			c := s.term.CellAt(x, y)
			// Keep a cell on a background
			if c != nil && c.Style.Bg != nil {
				run.WriteString(c.Content)
			}
		}
		// Keep a line that holds such cells
		if t := strings.TrimSpace(run.String()); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// settled is how long the screen stays the same before a test reads it as drawn.
const settled = 150 * time.Millisecond

// Settle waits until the screen stays the same for a moment, and reports whether it did within Wait.
func (s *Session) Settle() bool {
	last, since := s.Screen(), time.Now()
	return Within(func() bool {
		now := s.Screen()
		// Start the moment over where the screen changed
		if now != last {
			last, since = now, time.Now()
			return false
		}
		return time.Since(since) >= settled
	})
}
