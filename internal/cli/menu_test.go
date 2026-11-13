package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/config"
	"github.com/hansehart/o3s/internal/devcontainer"
	"github.com/hansehart/o3s/internal/home"
	"github.com/hansehart/o3s/internal/menu"
	"github.com/hansehart/o3s/internal/registry"
	"github.com/hansehart/o3s/internal/registry/registrytest"
)

// The menu knows every secret the home names or stores, whether it holds a value, and the home's stores.
func TestMenuHomeReadsTheHome(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	secretsDir := t.TempDir()
	t.Setenv("O3S_SECRETS_DIR", secretsDir)
	settings := `{"$schema": "./.devcontainer/o3s.schema.json", "stores": [{"id": "registry.example.com/features"}], "hosts": [
  {"name": "example.com", "ports": [443], "secret": "EXAMPLE_TOKEN", "inject": ["header"], "by": ["user"]}
]}`
	// Stop when the settings stay unwritten
	if err := os.WriteFile(filepath.Join(dir, "o3s.json"), []byte(settings), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Stop when the secrets stay unwritten
	if err := os.WriteFile(filepath.Join(secretsDir, "secrets.env"), []byte("OTHER_TOKEN=value\n"), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	got := menuHome(t.Context())

	got.Lookup = nil
	want := menu.Home{
		Dir:     dir,
		Hosts:   got.Hosts,
		Secrets: []menu.Secret{{Name: "EXAMPLE_TOKEN"}, {Name: "OTHER_TOKEN", Set: true}},
		Stores:  []string{"registry.example.com/features"},
	}
	// Report a home read otherwise
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("menuHome() (-want +got):\n%s", diff)
	}
	// Require the host the settings name
	if len(got.Hosts) != 1 || got.Hosts[0].Name != "example.com" {
		t.Errorf("menuHome() hosts = %v, want example.com", got.Hosts)
	}
}

// Outside a home the menu still starts, knowing only where it runs.
func TestMenuHomeStartsOutsideAHome(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("O3S_SECRETS_DIR", t.TempDir())

	got := menuHome(t.Context())

	got.Lookup = nil
	// Require a home that names the folder alone
	if diff := cmp.Diff(menu.Home{Dir: dir}, got); diff != "" {
		t.Errorf("menuHome() (-want +got):\n%s", diff)
	}
}

// aStoreWithTheTool raises a store that offers the tool and an other feature, and hands back the store.
func aStoreWithTheTool(t *testing.T) string {
	t.Helper()
	r, err := registrytest.New()
	// Stop when the registry fails to come up
	if err != nil {
		t.Fatalf("Setup: registrytest.New() = %v", err)
	}
	t.Cleanup(r.Close)
	tool := `{"id": "tool", "version": "1.0.0", "customizations": {"o3s": {
  "hosts": [{"name": "example.com", "ports": [443], "secret": "EXAMPLE_TOKEN"}],
  "env": [{"name": "EXAMPLE_MODE", "value": "strict"}]
}}}`
	// Stop when the store fails to publish
	if err := r.PushFeatures("features", tool, `{"id": "other"}`); err != nil {
		t.Fatalf("Setup: PushFeatures() = %v", err)
	}
	return r.Host + "/features"
}

// aHomeIn makes a new home the test works in, with secrets of its own.
func aHomeIn(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("O3S_SECRETS_DIR", t.TempDir())
	// Stop when the home stays unwritten
	if _, err := home.Init(dir); err != nil {
		t.Fatalf("Setup: Init() = %v", err)
	}
	return dir
}

// The menu reads a store's features and shows what adding one brings, as feature add prints it.
func TestMenuLookupShowsWhatAFeatureBrings(t *testing.T) {
	store := aStoreWithTheTool(t)
	aHomeIn(t)
	l := menuLookup{ctx: t.Context(), r: &registry.Resolver{}}

	ids, err := l.Features(store)
	// Stop when the store fails to read
	if err != nil {
		t.Fatalf("Features() = %v", err)
	}
	pv, err := l.Add(store + "/tool")
	// Stop when the preview fails
	if err != nil {
		t.Fatalf("Add() = %v", err)
	}

	// Require the store's ids in order
	if diff := cmp.Diff([]string{"other", "tool"}, ids); diff != "" {
		t.Errorf("Features() (-want +got):\n%s", diff)
	}
	want := menu.Preview{
		Lines:   []string{store + "/tool:1", "  + example.com:443  EXAMPLE_TOKEN ⚠", "  + EXAMPLE_MODE=strict"},
		Waiting: []string{"EXAMPLE_TOKEN"},
	}
	// Require the lines feature add prints
	if diff := cmp.Diff(want, pv); diff != "" {
		t.Errorf("Add() (-want +got):\n%s", diff)
	}
}

// The menu shows what leaves with an installed feature, and lists the features the home holds.
func TestMenuLookupShowsWhatLeavesWithAFeature(t *testing.T) {
	store := aStoreWithTheTool(t)
	dir := aHomeIn(t)
	pkg := store + "/tool"
	c, err := config.Load(dir)
	// Stop when the settings fail to load
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Stop when the tool's host stays out
	if err := c.Need(pkg, config.Change{Name: "example.com", Secret: "EXAMPLE_TOKEN"}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	dc, err := devcontainer.Load(dir)
	// Stop when the config fails to load
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	dc.SetFeature(pkg, pkg+":1", nil)
	dc.SetEnv("EXAMPLE_MODE", "strict")
	// Stop when the home fails to save
	if err := saveHome(dir, c, dc); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	l := menuLookup{ctx: t.Context(), r: &registry.Resolver{}}

	pv, err := l.Remove(pkg)

	// Stop when the preview fails
	if err != nil {
		t.Fatalf("Remove() = %v", err)
	}
	// Require the lines feature rm prints
	if diff := cmp.Diff(menu.Preview{Lines: []string{pkg, "  - example.com:443", "  - EXAMPLE_MODE=strict"}}, pv); diff != "" {
		t.Errorf("Remove() (-want +got):\n%s", diff)
	}
	// Require the installed feature on offer
	if diff := cmp.Diff([]string{pkg}, menuHome(t.Context()).Features); diff != "" {
		t.Errorf("menuHome().Features (-want +got):\n%s", diff)
	}
}

// scripted answers each question with the next answer, then steps back.
type scripted struct {
	answers [][]string
}

// Ask hands back the next answer, or a step back once the script ends.
func (s *scripted) Ask(menu.Question) ([]string, error) {
	// Step back once every answer is given
	if len(s.answers) == 0 {
		return nil, menu.ErrBack
	}
	a := s.answers[0]
	s.answers = s.answers[1:]
	return a, nil
}

// aSession is a session on a scripted user, recording each command, each wait and each read of the home.
func aSession(answers ...[]string) (*session, *[][]string, *int, *int) {
	var ran [][]string
	var pauses, reads int
	return &session{
		ask:   &scripted{answers: answers},
		pause: func() error { pauses++; return nil },
		home:  func() menu.Home { reads++; return menu.Home{Dir: "/home/ubuntu/o3s"} },
		run:   func(args []string, _ io.Reader) int { ran = append(ran, args); return 0 },
		out:   io.Discard,
	}, &ran, &pauses, &reads
}

// The menu stays open after each action, reads the home anew for each dialog, and closes on a step back.
func TestSessionRunsDialogsUntilTheUserLeaves(t *testing.T) {
	s, ran, pauses, reads := aSession([]string{"firewall"}, []string{"ls"}, []string{"store"}, []string{"ls"})

	code := s.loop()

	want := [][]string{{"firewall", "ls"}, {"store", "ls"}}
	// Report commands that differ
	if diff := cmp.Diff(want, *ran); diff != "" {
		t.Errorf("commands (-want +got):\n%s", diff)
	}
	// Require a wait after each action and a fresh home for each dialog
	if code != 0 || *pauses != 2 || *reads != 3 {
		t.Errorf("loop() = %d with %d waits and %d reads, want 0, 2, 3", code, *pauses, *reads)
	}
}

// The shell and the update may replace o3s with another process, so the session hands them the terminal and ends.
func TestSessionEndsWithACommandThatTakesOver(t *testing.T) {
	tests := [][][]string{
		{{"cage"}, {"shell"}, {"store"}, {"ls"}},
		{{"update"}, {"store"}, {"ls"}},
	}
	want := [][][]string{{{"cage", "shell"}}, {{"update"}}}
	// Run every dialog that takes over
	for i, answers := range tests {
		s, ran, pauses, _ := aSession(answers...)
		left := false
		s.leave = func() { left = true }

		s.loop()

		// Require the command as the last one, the screen handed back first, without a wait
		if diff := cmp.Diff(want[i], *ran); diff != "" || *pauses != 0 || !left {
			t.Errorf("commands (-want +got):\n%s, %d waits, left %v", diff, *pauses, left)
		}
	}
}
