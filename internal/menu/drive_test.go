package menu

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// step is one reply a scripted user gives: values, or an error for a key like Esc.
type step struct {
	values []string
	err    error
}

// script is a user who replies to each question in turn, and records what was asked.
type script struct {
	steps []step
	asked []string
}

// Ask replies with the next scripted step.
func (s *script) Ask(q Question) ([]string, error) {
	s.asked = append(s.asked, q.Title)
	// Leave once the script runs out, as Ctrl+C would
	if len(s.asked) > len(s.steps) {
		return nil, ErrQuit
	}
	st := s.steps[len(s.asked)-1]
	return st.values, st.err
}

// say is a scripted reply with values.
func say(values ...string) step { return step{values: values} }

// Esc steps back a question at a time and leaves the dialog for another.
func TestDriveStepsBackOnEsc(t *testing.T) {
	s := &script{steps: []step{say("firewall"), say("add"), {err: ErrBack}, {err: ErrBack}, say("store"), say("rm"), say("registry.example.com/features")}}

	got, ok, err := Drive(s, aHome)

	// Stop when the dialog fails
	if err != nil || !ok {
		t.Fatalf("Drive() = %v, %v, %v, want a command", got, ok, err)
	}
	// Require the command of the dialog chosen last
	if diff := cmp.Diff([]Command{{Args: []string{"store", "rm", "registry.example.com/features"}}}, got); diff != "" {
		t.Errorf("Drive() (-want +got):\n%s", diff)
	}
	want := []string{"What do you want to do?", "Action", "Host", "Action", "What do you want to do?", "Action", "Store"}
	// Require each question asked again after Esc
	if diff := cmp.Diff(want, s.asked); diff != "" {
		t.Errorf("asked (-want +got):\n%s", diff)
	}
}

// Esc on the first question leaves without a command.
func TestDriveLeavesOnEscAtTheStart(t *testing.T) {
	s := &script{steps: []step{{err: ErrBack}}}

	_, ok, err := Drive(s, aHome)

	// Require the menu to leave quietly
	if err != nil || ok {
		t.Errorf("Drive() = %v, %v, want to leave", ok, err)
	}
}

// Ctrl+C leaves at once, wherever the dialog stands.
func TestDriveLeavesOnCtrlC(t *testing.T) {
	s := &script{steps: []step{say("firewall"), say("add"), say("example.net"), {err: ErrQuit}}}

	_, ok, err := Drive(s, aHome)

	// Require the menu to leave quietly
	if err != nil || ok {
		t.Errorf("Drive() = %v, %v, want to leave", ok, err)
	}
}

// Esc returns to the question before, whose new answer may skip the one Esc left.
func TestDriveStepsBackToAnAnswerThatSkipsAQuestion(t *testing.T) {
	s := &script{steps: []step{say("firewall"), say("add"), say("example.net"), say("443"), say("OTHER_TOKEN"), {err: ErrBack}, say("none")}}

	got, ok, err := Drive(s, aHome)

	// Stop when the dialog fails
	if err != nil || !ok {
		t.Fatalf("Drive() = %v, %v, %v, want a command", got, ok, err)
	}
	// Require the command with the answer given after Esc
	if diff := cmp.Diff([]Command{{Args: []string{"firewall", "add", "example.net", "--port", "443"}}}, got); diff != "" {
		t.Errorf("Drive() (-want +got):\n%s", diff)
	}
	want := []string{"What do you want to do?", "Action", "Host", "Ports", "Secret", "Inject", "Secret"}
	// Require the secret asked again, and where it goes no more
	if diff := cmp.Diff(want, s.asked); diff != "" {
		t.Errorf("asked (-want +got):\n%s", diff)
	}
}

// Esc on a value returns to Apply?, and Esc there returns to the feature.
func TestDriveStepsBackThroughTheFeatureDialog(t *testing.T) {
	h := aHomeWithTheTool()
	s := &script{steps: []step{say("feature"), say("add"), say("registry.example.com/features"), say("tool"), say("yes"), {err: ErrBack}, {err: ErrBack}, say("other"), say("no")}}

	got, ok, err := Drive(s, h)

	// Require a declined dialog that runs nothing
	if err != nil || !ok || got != nil {
		t.Fatalf("Drive() = %v, %v, %v, want no command", got, ok, err)
	}
	want := []string{"What do you want to do?", "Action", "Store", "Feature", "Apply?", "Value for EXAMPLE_TOKEN (empty to skip)", "Apply?", "Feature", "Apply?"}
	// Require each question asked again after Esc
	if diff := cmp.Diff(want, s.asked); diff != "" {
		t.Errorf("asked (-want +got):\n%s", diff)
	}
}
