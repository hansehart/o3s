package menu

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/config"
)

// aHome is a home that allows a host with a secret, knows two secrets, and offers one store.
var aHome = Home{
	Dir: "/home/ubuntu/o3s",
	Hosts: []config.Host{
		{Name: "example.com", Ports: []int{22, 443}, By: []string{"user"}},
		{Name: "api.example.com", Ports: []int{443}, Secret: "EXAMPLE_TOKEN", Inject: []string{"header", "body"}, By: []string{"user"}},
		{Name: "example.org", Ports: []int{443}, By: []string{"registry.example.com/features/tool"}},
	},
	Secrets: []Secret{{Name: "EXAMPLE_TOKEN", Set: false}, {Name: "OTHER_TOKEN", Set: true}},
	Stores:  []string{"registry.example.com/features"},
}

// walk answers each question the core asks in turn and hands back what it asked and the command it ends in.
func walk(t *testing.T, h Home, answers ...[]string) ([]Question, []Command) {
	t.Helper()
	var a Answers
	var asked []Question
	// Answer every question until the dialog ends
	for {
		q, done := next(h, a)
		// End the walk once the dialog asks nothing more
		if done {
			return asked, plan(h, a)
		}
		// Stop when the dialog asks more than the test answers
		if len(asked) == len(answers) {
			t.Fatalf("walk: asked %q beyond the %d answers", q.Title, len(answers))
		}
		a = a.With(q.Key, answers[len(asked)])
		asked = append(asked, q)
	}
}

// titles lists what each question asked.
func titles(qs []Question) []string {
	out := make([]string, len(qs))
	// Take every question's title
	for i, q := range qs {
		out[i] = q.Title
	}
	return out
}

// Every dialog ends in the command line the CLI runs for it.
func TestEveryDialogEndsInItsCommand(t *testing.T) {
	value := "s3cr3t"
	tests := []struct {
		name    string
		answers [][]string
		titles  []string
		want    []Command
	}{
		{
			"a new host with a new secret and two sites",
			[][]string{{"firewall"}, {"add"}, {"example.net"}, {"443,8443"}, {"new"}, {"NEW_TOKEN"}, {"header", "body"}},
			[]string{"What do you want to do?", "Action", "Host", "Ports", "Secret", "Secret name", "Inject"},
			[]Command{{Args: []string{"firewall", "add", "example.net", "--port", "443,8443", "--secret", "NEW_TOKEN", "--inject", "header,body"}}},
		},
		{
			"a host without a secret skips where to inject it",
			[][]string{{"firewall"}, {"add"}, {"example.net"}, {"443"}, {"none"}},
			[]string{"What do you want to do?", "Action", "Host", "Ports", "Secret"},
			[]Command{{Args: []string{"firewall", "add", "example.net", "--port", "443"}}},
		},
		{
			"removing a host the user needs",
			[][]string{{"firewall"}, {"rm"}, {"example.com"}},
			[]string{"What do you want to do?", "Action", "Host"},
			[]Command{{Args: []string{"firewall", "rm", "example.com"}}},
		},
		{
			"listing the allowlist",
			[][]string{{"firewall"}, {"ls"}},
			[]string{"What do you want to do?", "Action"},
			[]Command{{Args: []string{"firewall", "ls"}}},
		},
		{
			"a value for a known secret",
			[][]string{{"secret"}, {"add"}, {"EXAMPLE_TOKEN"}, {value}},
			[]string{"What do you want to do?", "Action", "Name", "Value"},
			[]Command{{Args: []string{"secret", "add", "EXAMPLE_TOKEN"}, Stdin: &value}},
		},
		{
			"a value for a secret of another name",
			[][]string{{"secret"}, {"add"}, {"other"}, {"NEW_TOKEN"}, {value}},
			[]string{"What do you want to do?", "Action", "Name", "Secret name", "Value"},
			[]Command{{Args: []string{"secret", "add", "NEW_TOKEN"}, Stdin: &value}},
		},
		{
			"removing a stored secret",
			[][]string{{"secret"}, {"rm"}, {"OTHER_TOKEN"}},
			[]string{"What do you want to do?", "Action", "Name"},
			[]Command{{Args: []string{"secret", "rm", "OTHER_TOKEN"}}},
		},
		{
			"adding a store",
			[][]string{{"store"}, {"add"}, {"registry.example.org/features"}},
			[]string{"What do you want to do?", "Action", "Store"},
			[]Command{{Args: []string{"store", "add", "registry.example.org/features"}}},
		},
		{
			"removing a store",
			[][]string{{"store"}, {"rm"}, {"registry.example.com/features"}},
			[]string{"What do you want to do?", "Action", "Store"},
			[]Command{{Args: []string{"store", "rm", "registry.example.com/features"}}},
		},
		{
			"creating a home",
			[][]string{{"init"}, {"yes"}},
			[]string{"What do you want to do?", "Create an o3s home in /home/ubuntu/o3s?"},
			[]Command{{Args: []string{"init"}}},
		},
		{
			"declining to create a home",
			[][]string{{"init"}, {"no"}},
			[]string{"What do you want to do?", "Create an o3s home in /home/ubuntu/o3s?"},
			nil,
		},
		{
			"updating the home",
			[][]string{{"update"}},
			[]string{"What do you want to do?"},
			[]Command{{Args: []string{"update"}}},
		},
	}
	// Walk every dialog
	for _, tt := range tests {
		asked, got := walk(t, aHome, tt.answers...)

		// Report questions that differ
		if diff := cmp.Diff(tt.titles, titles(asked)); diff != "" {
			t.Errorf("%s: questions (-want +got):\n%s", tt.name, diff)
		}
		// Report a command that differs
		if diff := cmp.Diff(tt.want, got); diff != "" {
			t.Errorf("%s: command (-want +got):\n%s", tt.name, diff)
		}
	}
}

// A host the allowlist holds offers its ports, its secret and its sites as the defaults.
func TestFirewallAddOffersTheHostsCurrentValues(t *testing.T) {
	asked, _ := walk(t, aHome, []string{"firewall"}, []string{"add"}, []string{"api.example.com"}, []string{"443"}, []string{"EXAMPLE_TOKEN"}, []string{"header", "body"})

	got := []any{asked[3].Default, asked[4].Default, asked[5].Default}

	want := []any{[]string{"443"}, []string{"EXAMPLE_TOKEN"}, []string{"header", "body"}}
	// Report defaults that differ
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("defaults (-want +got):\n%s", diff)
	}
}

// A new host starts on the HTTPS port and puts a secret into the header.
func TestFirewallAddOffersTheDefaultsOfANewHost(t *testing.T) {
	asked, _ := walk(t, aHome, []string{"firewall"}, []string{"add"}, []string{"example.net"}, []string{"443"}, []string{"OTHER_TOKEN"}, []string{"header"})

	got := []any{asked[3].Default, asked[4].Default, asked[5].Default}

	want := []any{[]string{"443"}, []string{"none"}, []string{"header"}}
	// Report defaults that differ
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("defaults (-want +got):\n%s", diff)
	}
}

// The secrets on offer show whether each holds a value, and leave room for none or a new one.
func TestFirewallAddOffersEverySecret(t *testing.T) {
	asked, _ := walk(t, aHome, []string{"firewall"}, []string{"add"}, []string{"example.net"}, []string{"443"}, []string{"none"})

	got := asked[4].Options

	want := []Option{{Value: "none"}, {Value: "EXAMPLE_TOKEN", Note: "empty"}, {Value: "OTHER_TOKEN", Note: "set"}, {Value: "new"}}
	// Report options that differ
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("options (-want +got):\n%s", diff)
	}
}

// Removing offers the hosts the user needs alone, since features own the rest.
func TestFirewallRmOffersTheUsersHosts(t *testing.T) {
	asked, _ := walk(t, aHome, []string{"firewall"}, []string{"rm"}, []string{"example.com"})

	got := asked[2].Options

	want := []Option{{Value: "example.com"}, {Value: "api.example.com"}}
	// Report options that differ
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("options (-want +got):\n%s", diff)
	}
}

// A removal with nothing to remove ends with a note.
func TestRmWithNothingToRemoveEndsWithANote(t *testing.T) {
	tests := []struct {
		group string
		want  string
	}{
		{"firewall", "No host to remove"},
		{"secret", "No secret to remove"},
		{"store", "No store to remove"},
	}
	// Check every group
	for _, tt := range tests {
		_, got := walk(t, Home{Dir: "/home/ubuntu/o3s"}, []string{tt.group}, []string{"rm"})

		// Report a command that differs
		if diff := cmp.Diff([]Command{{Note: tt.want}}, got); diff != "" {
			t.Errorf("%s rm (-want +got):\n%s", tt.group, diff)
		}
	}
}
