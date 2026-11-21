//go:build e2e

// Package usage_test uses o3s the way a user does, on the real stack: the commands that change a home in orders that
// hold every three of them in every order, and every path of the menu in a real terminal, on every setup of
// usagetest's table, with a failure halfway through where the setup names one. Every output and every screen keeps
// the rules of screentest, and every command that succeeds leaves the home the way it says.
package usage_test

import (
	"context"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hansehart/o3s/internal/cli"
	"github.com/hansehart/o3s/internal/fixture"
	"github.com/hansehart/o3s/internal/menu/menutest"
	"github.com/hansehart/o3s/internal/ptytest"
	"github.com/hansehart/o3s/internal/tui/screentest"
	"github.com/hansehart/o3s/test/stacktest"
	"github.com/hansehart/o3s/test/usagetest"
)

// knownBreaks are the breaks o3s shows today, each by where it shows and the rule it breaks, with what a fix changes.
// A fix removes its entry, and a new break fails the test.
var knownBreaks = map[string]string{
	"add a feature: no next step":                                 "a feature add with the registry out of reach names no command to run next",
	"add a project over SSH: devcontainer log":                    "a project add shows the devcontainer CLI's own log",
	"add a project over SSH: no clone in the cage":                "a project add over SSH says Added with no clone in the cage",
	"add a project over SSH: no next step":                        "a project add over SSH that fails names no command that works",
	"add a project over SSH: the cage misses the session's agent": "the cage misses the SSH agent the user's session holds",
	"add a store: no next step":                                   "a store add with the registry out of reach names no command to run next",
	"close a host: no next step":                                  "closing a host the firewall lacks names no command to run next",
	"menu feature / Discover [] / no: no footer":                  "the Discover tab runs past a short terminal, its footer out of sight",
	"menu feature / Discover [] / yes: no footer":                 "the Discover tab runs past a short terminal, its footer out of sight",
	"remove a feature: no next step":                              "removing a feature the home lacks names no command to run next",
	"remove a project: devcontainer log":                          "a project rm shows the devcontainer CLI's own log",
	"remove a secret: no next step":                               "removing a secret the home lacks names no command to run next",
	"remove a store: no next step":                                "removing a store the home lacks names no command to run next",
	"start the cage: devcontainer log":                            "a cage start shows the devcontainer CLI's own log",
	"start the cage: hangs":                                       "a cage start without the network waits past any working start",
}

// Real things the commands reach, each a value the test checks itself.
const (
	store   = "ghcr.io/devcontainers/features"
	feature = store + "/node"
	repo    = "git@github.com:octocat/Hello-World.git"
	clone   = "/home/ubuntu/projects/github.com/octocat/Hello-World"
)

// user is a user of the test's home on one setup: where the terminal is, the agent it holds, and the failure in place.
type user struct {
	t     *testing.T
	setup usagetest.Setup
	home  string
	agent agent
	// unreachable sends o3s's own requests to a closed port, the way a registry out of reach does.
	unreachable bool
}

// env is the environment o3s runs in for the user: the test's, the agent the session holds, and the proxy of a
// registry out of reach.
func (u *user) env() []string {
	out := slices.DeleteFunc(os.Environ(), func(v string) bool {
		return strings.HasPrefix(v, "SSH_AUTH_SOCK=") || strings.HasPrefix(v, "HTTPS_PROXY=")
	})
	// Hand o3s the agent the session holds
	if u.agent.sock != "" {
		out = append(out, "SSH_AUTH_SOCK="+u.agent.sock)
	}
	// Send o3s's requests to a closed port where the registry is out of reach
	if u.unreachable {
		out = append(out, "HTTPS_PROXY=http://127.0.0.1:9")
	}
	return out
}

// command is how the user's terminal starts o3s with the arguments given: on the machine, or over SSH with the agent
// forwarded where the session holds one, with a terminal at the other end where o3s runs in one.
func (u *user) command(tty bool, args ...string) (string, []string) {
	bin, err := exec.LookPath("o3s")
	// Stop when the binary this branch built is off PATH
	if err != nil {
		u.t.Fatalf("Setup: find o3s: %v", err)
	}
	// Start o3s on the machine
	if u.setup.Session == "local" {
		return bin, args
	}
	var vars []string
	// Carry over what o3s reads from its environment, the session's agent left to SSH
	for _, v := range u.env() {
		name, _, _ := strings.Cut(v, "=")
		// Carry a variable o3s reads
		if slices.Contains(carried, name) || strings.HasPrefix(name, "XDG_") || strings.HasPrefix(name, "O3S_") {
			vars = append(vars, quote(v))
		}
	}
	line := "cd " + quote(u.home) + " && env " + strings.Join(vars, " ") + " " + quote(bin)
	// Quote every argument for the remote shell
	for _, a := range args {
		line += " " + quote(a)
	}
	ssh := []string{"-T", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new"}
	// Give o3s a terminal at the other end where it runs in one here
	if tty {
		ssh[0] = "-tt"
	}
	// Forward the agent the session holds
	if u.agent.sock != "" {
		ssh = append(ssh, "-A")
	}
	return "ssh", append(ssh, "localhost", line)
}

// carried are the variables a session over SSH carries to o3s beside its own XDG_ and O3S_ ones.
var carried = []string{"PATH", "DOCKER_HOST", "HTTPS_PROXY", "BUILDX_BAKE_ENTITLEMENTS_FS"}

// quote quotes a word for a POSIX shell.
func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// patience is how long a command runs before the test reads it as hanging, well past what any command takes on a
// working home.
const patience = 30 * time.Second

// o3s runs o3s in the user's session with the input given, the way a script runs it, and hands back what it printed,
// whether it failed, and whether it hung past the test's patience.
func (u *user) o3s(stdin string, args ...string) (string, bool, bool) {
	ctx, cancel := context.WithTimeout(u.t.Context(), patience)
	defer cancel()
	name, argv := u.command(false, args...)
	cmd := exec.CommandContext(ctx, name, argv...)
	cmd.Dir, cmd.Env, cmd.Stdin = u.home, u.env(), strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	return string(out), err != nil, ctx.Err() != nil
}

// cageRuns reports whether the cage's container runs.
func (u *user) cageRuns() bool {
	out, err := exec.CommandContext(u.t.Context(), "docker", "ps", "-q",
		"--filter", "label=com.docker.compose.project.working_dir="+filepath.Join(u.home, ".devcontainer"),
		"--filter", "label=com.docker.compose.service=cage").Output()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

// inCage runs a command in the cage the way o3s runs it there, and reports whether it succeeded within the test's
// patience.
func (u *user) inCage(command ...string) bool {
	ctx, cancel := context.WithTimeout(u.t.Context(), patience)
	defer cancel()
	cmd := exec.CommandContext(ctx, "devcontainer", append([]string{"exec", "--workspace-folder", u.home}, command...)...)
	cmd.Env = u.env()
	return cmd.Run() == nil
}

// lists reports whether a group's table lists a row that starts with the name.
func (u *user) lists(name string, group ...string) bool {
	out, _, _ := u.o3s("", group...)
	return slices.ContainsFunc(strings.Split(out, "\n"), func(line string) bool { return strings.HasPrefix(line, name+" ") })
}

// agent is an SSH agent a session holds, with a key of its own.
type agent struct {
	sock string
	pid  string
}

// anAgent starts an SSH agent holding a fresh key, the way a developer's session holds one.
func anAgent(t *testing.T) agent {
	t.Helper()
	dir := t.TempDir()
	key, sock := filepath.Join(dir, "id_ed25519"), filepath.Join(dir, "agent.sock")
	// Stop when the key stays unmade
	if out, err := exec.CommandContext(t.Context(), "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key).CombinedOutput(); err != nil {
		t.Fatalf("Setup: ssh-keygen: %v\n%s", err, out)
	}
	out, err := exec.CommandContext(t.Context(), "ssh-agent", "-s", "-a", sock).Output()
	// Stop when the agent stays down
	if err != nil {
		t.Fatalf("Setup: ssh-agent: %v", err)
	}
	a := agent{sock: sock}
	// Read the agent's process from what it prints for a shell
	for _, field := range strings.FieldsFunc(string(out), func(r rune) bool { return r == ';' || r == '\n' }) {
		// Take the agent's process
		if pid, ok := strings.CutPrefix(strings.TrimSpace(field), "SSH_AGENT_PID="); ok {
			a.pid = pid
		}
	}
	t.Cleanup(func() { a.stop() })
	add := exec.CommandContext(t.Context(), "ssh-add", key)
	add.Env = append(os.Environ(), "SSH_AUTH_SOCK="+sock)
	// Stop when the agent turns the key away
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("Setup: ssh-add: %v\n%s", err, out)
	}
	return a
}

// stop ends the agent.
func (a agent) stop() {
	// Leave an agent that never started
	if a.pid == "" {
		return
	}
	pid, err := strconv.Atoi(a.pid)
	// Leave an agent whose process reads as none
	if err != nil {
		return
	}
	p, err := os.FindProcess(pid)
	// Leave an agent whose process is gone
	if err != nil {
		return
	}
	_ = p.Kill()
}

// event is a command that changes the home, and the check that the home is the way the command says.
type event struct {
	name  string
	stdin string
	args  []string
	// holds is what differs from the home the command left, empty where it is the way the command says.
	holds func(u *user) string
}

// events are every command the orders run.
var events = []event{
	{name: "allow a host", args: []string{"firewall", "add", "github.com", "--port", "22"}, holds: func(u *user) string {
		return unless(u.lists("github.com", "firewall"), "o3s firewall lists no github.com")
	}},
	{name: "close a host", args: []string{"firewall", "rm", "github.com", "--yes"}, holds: func(u *user) string {
		return unless(!u.lists("github.com", "firewall"), "o3s firewall still lists github.com")
	}},
	{name: "add a store", args: []string{"feature", "store", "add", store}, holds: func(u *user) string {
		return unless(u.lists(store, "feature", "store"), "o3s feature store lists no "+store)
	}},
	{name: "remove a store", args: []string{"feature", "store", "rm", store, "--yes"}, holds: func(u *user) string {
		return unless(!u.lists(store, "feature", "store"), "o3s feature store still lists "+store)
	}},
	{name: "add a feature", args: []string{"feature", "add", feature}, holds: func(u *user) string {
		return unless(u.lists("node", "feature"), "o3s feature lists no node")
	}},
	{name: "remove a feature", args: []string{"feature", "rm", feature, "--yes"}, holds: func(u *user) string {
		return unless(!u.lists("node", "feature"), "o3s feature still lists node")
	}},
	{name: "add a secret", stdin: fixture.SecretValue, args: []string{"secret", "add", fixture.Secret}, holds: func(u *user) string {
		return unless(u.lists(fixture.Secret, "secret"), "o3s secret lists no "+fixture.Secret)
	}},
	{name: "remove a secret", args: []string{"secret", "rm", fixture.Secret, "--yes"}, holds: func(u *user) string {
		return unless(!u.lists(fixture.Secret, "secret"), "o3s secret still lists "+fixture.Secret)
	}},
	{name: "add a project over SSH", args: []string{"project", "add", repo}, holds: func(u *user) string {
		cloned := u.inCage("test", "-f", clone+"/README")
		return unless(cloned, "no clone in the cage")
	}},
	{name: "remove a project", args: []string{"project", "rm", "octocat/Hello-World", "--yes"}, holds: func(u *user) string {
		left := u.inCage("test", "-e", clone)
		return unless(!left, "the clone stays in the cage")
	}},
	{name: "stop the cage", args: []string{"cage", "stop"}, holds: func(u *user) string {
		running := u.cageRuns()
		return unless(!running, "the cage still runs")
	}},
	{name: "start the cage", args: []string{"cage", "start"}, holds: func(u *user) string {
		running := u.cageRuns()
		return unless(running, "the cage runs not")
	}},
}

// unless is the difference given where the condition fails, and none where it holds.
func unless(ok bool, differs string) string {
	// Name no difference where the condition holds
	if ok {
		return ""
	}
	return differs
}

// fault is a failure the setup names, with how it starts and how it ends.
type fault struct {
	start, end func(u *user)
}

// faults are the failures the setups name, by name.
var faults = map[string]fault{
	"none": {start: func(*user) {}, end: func(*user) {}},
	"proxy-down": {
		start: func(u *user) { _, _ = stacktest.Compose(u.t, "stop", "proxy") },
		end:   func(u *user) { _, _ = stacktest.Compose(u.t, "start", "proxy") },
	},
	"network-cut": {
		start: func(u *user) {
			_ = exec.CommandContext(u.t.Context(), "sudo", "iptables", "-I", "DOCKER-USER", "-j", "REJECT").Run()
		},
		end: func(u *user) {
			_ = exec.CommandContext(u.t.Context(), "sudo", "iptables", "-D", "DOCKER-USER", "-j", "REJECT").Run()
		},
	},
	"registry-down": {
		start: func(u *user) { u.unreachable = true },
		end:   func(u *user) { u.unreachable = false },
	},
	"agent-gone": {
		start: func(u *user) { u.agent.stop() },
		end:   func(u *user) { u.agent = anAgent(u.t) },
	},
}

// found gathers the breaks a setup shows, each once, by where it shows and the rule it breaks.
type found map[string]bool

// note keeps every break given, by where it showed, and logs what showed it the first time.
func (f found) note(t *testing.T, where, shown string, breaks ...string) {
	t.Helper()
	// Keep every break
	for _, b := range breaks {
		key := where + ": " + b
		// Log what a break shows the first time
		if !f[key] {
			t.Logf("%s, on:\n%s", key, shown)
		}
		f[key] = true
	}
}

// Using o3s on every setup keeps every output and every screen to the rules and leaves the home the way each command
// says, outside the known breaks.
func TestUsage(t *testing.T) {
	setups, err := usagetest.Setups()
	// Stop where the table stays unread
	if err != nil {
		t.Fatalf("Setups() = %v", err)
	}
	name := os.Getenv("O3S_SETUP")
	// Stop where the job names a setup the table lacks
	if name != "" && !slices.ContainsFunc(setups, func(s usagetest.Setup) bool { return s.Name() == name }) {
		var names []string
		// Name every setup the table holds
		for _, s := range setups {
			names = append(names, s.Name())
		}
		t.Fatalf("O3S_SETUP=%s names no setup, want one of %s", name, strings.Join(names, ", "))
	}
	// Use o3s on every setup, or on the one the job names
	for _, s := range setups {
		// Leave the setups another job runs
		if name != "" && s.Name() != name {
			continue
		}
		t.Run(s.Name(), func(t *testing.T) {
			f := use(t, s)

			// Report every break no finding names
			for _, b := range slices.Sorted(maps.Keys(f)) {
				// Report a break outside the known ones
				if _, known := knownBreaks[b]; !known {
					t.Errorf("%s", b)
				}
			}
		})
	}
}

// use uses o3s on one setup: the commands in every order, then every path of the menu, and hands back the breaks it
// found.
func use(t *testing.T, s usagetest.Setup) found {
	t.Helper()
	home := stacktest.Home(t)
	u := &user{t: t, setup: s, home: home}
	// Hold an agent where the setup's session does
	if s.Agent == "host" {
		u.agent = anAgent(t)
	}
	// Stop when the hosts the commands reach stay closed
	if _, err := stacktest.O3s(t, "firewall", "add", "github.com", "--port", "443"); err != nil {
		t.Fatalf("Setup: o3s firewall add = %v", err)
	}
	// Stop when the stack fails to come up
	if _, err := stacktest.DevcontainerUp(t); err != nil {
		t.Fatalf("Setup: DevcontainerUp() = %v", err)
	}
	f := found{}
	runOrders(u, f)
	walkMenu(u, f)
	return f
}

// runOrders runs the events in every order, the setup's failure starting halfway through each, and notes every rule
// an output breaks and every claim the home contradicts.
func runOrders(u *user, f found) {
	fl := faults[u.setup.Fault]
	// Run every order
	for _, order := range usagetest.Orders(len(events)) {
		runOrder(u, f, fl, order)
		fl.end(u)
	}
}

// runOrder runs the events of one order, the failure starting halfway through, and ends it at the first command that
// hangs, since the ones after it would wait on the same failure.
func runOrder(u *user, f found, fl fault, order []int) {
	// Run every event of the order
	for i, e := range order {
		// Start the failure halfway through
		if i == len(order)/2 {
			fl.start(u)
		}
		ev := events[e]
		began := time.Now()
		out, failed, hung := u.o3s(ev.stdin, ev.args...)
		u.t.Logf("took %v: %s", time.Since(began).Round(time.Second), ev.name)
		// Note a command that hangs, and end the order
		if hung {
			f.note(u.t, ev.name, out, "hangs")
			return
		}
		f.note(u.t, ev.name, out, screentest.OutputBreaks(out, failed)...)
		// Check the home is the way a command that succeeded says
		if !failed {
			// Note a claim the home contradicts
			if d := ev.holds(u); d != "" {
				f.note(u.t, ev.name, out, d)
			}
		}
		// Check the cage reaches the agent the session holds, where the project needs it
		if ev.name == "add a project over SSH" && u.agent.pid != "" {
			// Note a cage without the session's agent
			if !u.inCage("ssh-add", "-l") {
				f.note(u.t, ev.name, out, "the cage misses the session's agent")
			}
		}
	}
}

// walkMenu walks every path of the home's menu in the user's terminal at the setup's size, up to the answer that
// would change the home, and notes every rule a screen breaks, a row out of sight among them.
func walkMenu(u *user, f found) {
	t := u.t
	// Raise the cage the orders may have left stopped
	if _, failed, _ := u.o3s("", "cage", "start"); failed {
		t.Fatalf("Setup: o3s cage start failed")
	}
	t.Chdir(u.home)
	paths, err := menutest.Walk(cli.MenuHome(t.Context()))
	// Stop where the menu can't be walked
	if err != nil {
		t.Fatalf("menutest.Walk() = %v", err)
	}
	// Walk every path
	for _, p := range paths {
		where := "menu " + strings.Split(menutest.Line(p), " → ")[0]
		name, args := u.command(true)
		s, err := ptytest.Start(u.home, u.env(), u.setup.Cols, u.setup.Rows, name, args...)
		// Stop where o3s fails to start
		if err != nil {
			t.Fatalf("ptytest.Start() = %v", err)
		}
		// Stop where o3s draws no menu in the user's terminal
		if !s.Until(func(screen string) bool { return strings.Contains(screen, "o3s  ") }) {
			t.Fatalf("o3s drew no menu within %v, on:\n%s", ptytest.Wait(), s.Screen())
		}
		s.Settle()
		// Answer every question up to the last, checking every screen
		for j, a := range p.Answers[:max(len(p.Answers)-1, 0)] {
			f.note(t, where, s.Screen(), screentest.Breaks(strings.Split(s.Screen(), "\n"), u.setup.Cols, u.setup.Rows)...)
			// Note a path whose rows a user can't reach
			if err := usagetest.Answer(s, p.Asked[j], a.Values); err != nil {
				f.note(t, where, s.Screen(), "row out of sight")
				break
			}
		}
		f.note(t, where, s.Screen(), screentest.Breaks(strings.Split(s.Screen(), "\n"), u.setup.Cols, u.setup.Rows)...)
		s.Stop()
	}
}
