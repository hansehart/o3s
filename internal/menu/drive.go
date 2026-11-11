package menu

import "errors"

var (
	// ErrBack marks a question the user stepped back from.
	ErrBack = errors.New("back")
	// ErrQuit marks a menu the user left at once.
	ErrQuit = errors.New("quit")
)

// Prompter asks the user one question and hands back the answer.
type Prompter interface {
	Ask(Question) ([]string, error)
}

// Drive leads the user through a dialog and hands back its command, or reports that the user left.
func Drive(p Prompter, h Home) (Command, bool, error) {
	var a Answers
	// Ask until the dialog ends or the user leaves
	for {
		q, done := next(h, a)
		// Build the command once the dialog asks nothing more
		if done {
			return plan(a), true, nil
		}
		values, err := p.Ask(q)
		// Follow the answer, a step back, or the user leaving
		switch {
		case errors.Is(err, ErrBack) && len(a) == 0, errors.Is(err, ErrQuit):
			return Command{}, false, nil
		case errors.Is(err, ErrBack):
			a = a[:len(a)-1]
		case err != nil:
			return Command{}, false, err
		default:
			a = a.With(q.Key, values)
		}
	}
}
