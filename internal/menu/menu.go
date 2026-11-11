// Package menu leads through the dialogs of bare o3s, each ending in the command line the CLI runs.
package menu

import (
	"slices"
	"strconv"
	"strings"

	"github.com/hansehart/o3s/internal/config"
)

// Home is what the menu knows about the home it runs in when it starts.
type Home struct {
	Dir     string
	Hosts   []config.Host
	Secrets []Secret
	Stores  []string
}

// Secret is a secret the home knows, and whether it holds a value.
type Secret struct {
	Name string
	Set  bool
}

// Kind is the way a question takes its answer.
type Kind int

const (
	// Select takes one of the options.
	Select Kind = iota
	// Multi takes any number of the options.
	Multi
	// Input takes a line of text.
	Input
	// Hidden takes a line of text the screen keeps hidden.
	Hidden
	// Confirm takes yes or no.
	Confirm
)

// Question is one step of a dialog.
type Question struct {
	Key     string
	Kind    Kind
	Title   string
	Options []Option
	Default []string
}

// Option is one choice of a question, with a note shown beside it.
type Option struct {
	Value string
	Note  string
}

// Answer is what the user gave one question.
type Answer struct {
	Key    string
	Values []string
}

// Answers are the answers given so far, in the order the dialog asked.
type Answers []Answer

// With hands back the answers with one more.
func (a Answers) With(key string, values []string) Answers {
	return append(slices.Clone(a), Answer{Key: key, Values: values})
}

// has reports whether a question was answered.
func (a Answers) has(key string) bool {
	return slices.ContainsFunc(a, func(x Answer) bool { return x.Key == key })
}

// get hands back the first value given a question.
func (a Answers) get(key string) string {
	values := a.all(key)
	// Read a question without a value as empty
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// all hands back every value given a question.
func (a Answers) all(key string) []string {
	i := slices.IndexFunc(a, func(x Answer) bool { return x.Key == key })
	// Read an unanswered question as empty
	if i < 0 {
		return nil
	}
	return a[i].Values
}

// Command is the command line a dialog ends in, the value it hands on stdin, and a note where it runs none.
type Command struct {
	Args  []string
	Stdin *string
	Note  string
}

// Answers the dialogs give.
const (
	none  = "none"
	fresh = "new"
	other = "other"
	yes   = "yes"
)

// next hands back the question a dialog asks next, or reports that it asks nothing more.
func next(h Home, a Answers) (Question, bool) {
	// Ask what to do first
	if !a.has("do") {
		return choose("do", "What do you want to do?", "init", "firewall", "secret", "store", "update"), false
	}
	do := a.get("do")
	// Ask for the confirmation a new home needs
	if do == "init" {
		return ask(a, Question{Key: "init", Kind: Confirm, Title: "Create an o3s home in " + h.Dir + "?"})
	}
	// Run the update without a further question
	if do == "update" {
		return Question{}, true
	}
	// Ask which action of the group comes next
	if !a.has("action") {
		return choose("action", "Action", "add", "rm", "ls"), false
	}
	action := a.get("action")
	// List without a further question
	if action == "ls" {
		return Question{}, true
	}
	// Follow the dialog of the group
	switch do {
	case "firewall":
		return firewall(h, a, action)
	case "secret":
		return secret(h, a, action)
	default:
		return store(h, a, action)
	}
}

// firewall asks what adding or removing a host needs.
func firewall(h Home, a Answers, action string) (Question, bool) {
	// Ask which of the user's hosts to remove
	if action == "rm" {
		return pick(a, "host", "Host", userHosts(h))
	}
	// Ask which host to allow
	if !a.has("host") {
		return Question{Key: "host", Kind: Input, Title: "Host"}, false
	}
	current, known := (&config.Config{Hosts: h.Hosts}).Find(a.get("host"))
	// Ask for the ports, offering the host's own or the HTTPS port
	if !a.has("ports") {
		ports := "443"
		// Offer the ports the host opens already
		if known {
			ports = joinInts(current.Ports)
		}
		return Question{Key: "ports", Kind: Input, Title: "Ports", Default: []string{ports}}, false
	}
	// Ask for the secret, offering the host's own or none
	if !a.has("secret") {
		q := Question{Key: "secret", Kind: Select, Title: "Secret", Options: []Option{{Value: none}}, Default: []string{none}}
		q.Options = append(q.Options, secretOptions(h)...)
		q.Options = append(q.Options, Option{Value: fresh})
		// Offer the secret the host carries already
		if known && current.Secret != "" {
			q.Default = []string{current.Secret}
		}
		return q, false
	}
	// Ask for the name of a new secret
	if a.get("secret") == fresh && !a.has("secretName") {
		return Question{Key: "secretName", Kind: Input, Title: "Secret name"}, false
	}
	// Ask where the secret goes, offering the host's own sites or the header
	if firewallSecret(a) != "" && !a.has("inject") {
		sites := []string{"header"}
		// Offer the sites the host uses already
		if known && len(current.Inject) > 0 {
			sites = current.Inject
		}
		return Question{Key: "inject", Kind: Multi, Title: "Inject", Options: []Option{{Value: "header"}, {Value: "query"}, {Value: "body"}}, Default: sites}, false
	}
	return Question{}, true
}

// secret asks what storing or removing a secret needs.
func secret(h Home, a Answers, action string) (Question, bool) {
	// Ask which stored secret to remove
	if action == "rm" {
		var stored []string
		// Offer every secret that holds a value
		for _, s := range h.Secrets {
			// Take a secret with a value
			if s.Set {
				stored = append(stored, s.Name)
			}
		}
		return pick(a, "name", "Name", stored)
	}
	// Ask which secret to store
	if !a.has("name") {
		options := append(secretOptions(h), Option{Value: other})
		return Question{Key: "name", Kind: Select, Title: "Name", Options: options}, false
	}
	// Ask for the name of another secret
	if a.get("name") == other && !a.has("secretName") {
		return Question{Key: "secretName", Kind: Input, Title: "Secret name"}, false
	}
	return ask(a, Question{Key: "value", Kind: Hidden, Title: "Value"})
}

// store asks what adding or removing a store needs.
func store(h Home, a Answers, action string) (Question, bool) {
	// Ask which store to remove
	if action == "rm" {
		return pick(a, "store", "Store", h.Stores)
	}
	return ask(a, Question{Key: "store", Kind: Input, Title: "Store"})
}

// plan builds the command line the answers lead to.
func plan(a Answers) Command {
	do, action := a.get("do"), a.get("action")
	// Build the command of the dialog
	switch {
	case do == "init":
		// Leave when the user declines
		if a.get("init") != yes {
			return Command{}
		}
		return Command{Args: []string{"init"}}
	case do == "update":
		return Command{Args: []string{"update"}}
	case action == "ls":
		return Command{Args: []string{do, "ls"}}
	case action == "rm":
		return remove(a, do)
	case do == "firewall":
		args := []string{"firewall", "add", a.get("host"), "--port", a.get("ports")}
		// Name the secret and where it goes
		if s := firewallSecret(a); s != "" {
			args = append(args, "--secret", s, "--inject", strings.Join(a.all("inject"), ","))
		}
		return Command{Args: args}
	case do == "secret":
		name := a.get("name")
		// Take the name the user typed for another secret
		if name == other {
			name = a.get("secretName")
		}
		value := a.get("value")
		return Command{Args: []string{"secret", "add", name}, Stdin: &value}
	default:
		return Command{Args: []string{"store", "add", a.get("store")}}
	}
}

// remove builds the removal a dialog chose, or the note that nothing was there to remove.
func remove(a Answers, do string) Command {
	keys := map[string]string{"firewall": "host", "secret": "name", "store": "store"}
	nouns := map[string]string{"firewall": "host", "secret": "secret", "store": "store"}
	// Name what was missing when there was nothing to choose
	if !a.has(keys[do]) {
		return Command{Note: "No " + nouns[do] + " to remove"}
	}
	return Command{Args: []string{do, "rm", a.get(keys[do])}}
}

// firewallSecret hands back the secret a host is to carry, and empty for none.
func firewallSecret(a Answers) string {
	s := a.get("secret")
	// Take the name the user typed for a new secret
	if s == fresh {
		return a.get("secretName")
	}
	// Read none as no secret
	if s == none {
		return ""
	}
	return s
}

// choose builds a question that takes one of plain options.
func choose(key, title string, values ...string) Question {
	q := Question{Key: key, Kind: Select, Title: title}
	// Offer every value
	for _, v := range values {
		q.Options = append(q.Options, Option{Value: v})
	}
	return q
}

// pick asks for one of the values once, and ends the dialog where there is nothing to choose.
func pick(a Answers, key, title string, values []string) (Question, bool) {
	// End the dialog where nothing is there to choose
	if len(values) == 0 {
		return Question{}, true
	}
	return ask(a, choose(key, title, values...))
}

// ask hands back a question until it is answered, and then ends the dialog.
func ask(a Answers, q Question) (Question, bool) {
	// End the dialog once the question is answered
	if a.has(q.Key) {
		return Question{}, true
	}
	return q, false
}

// secretOptions offers every secret the home knows, noting whether it holds a value.
func secretOptions(h Home) []Option {
	var out []Option
	// Offer every secret with its state
	for _, s := range h.Secrets {
		note := "empty"
		// Note a secret that holds a value
		if s.Set {
			note = "set"
		}
		out = append(out, Option{Value: s.Name, Note: note})
	}
	return out
}

// userHosts lists the hosts the user needs, which removing may drop.
func userHosts(h Home) []string {
	var out []string
	// Look at every host
	for _, host := range h.Hosts {
		// Take a host the user added
		if slices.Contains(host.By, "user") {
			out = append(out, host.Name)
		}
	}
	return out
}

// joinInts writes numbers as one comma-separated value.
func joinInts(ns []int) string {
	parts := make([]string, len(ns))
	// Write every number
	for i, n := range ns {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ",")
}
