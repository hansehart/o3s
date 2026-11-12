package menu

import "testing"

// Each answered question leaves the line #123's dialogs show for it.
func TestCollapseLeavesTheDialogsLine(t *testing.T) {
	tests := []struct {
		q      Question
		answer []string
		want   string
	}{
		{Question{Kind: Select, Title: "What do you want to do?"}, []string{"firewall"}, "> What do you want to do? firewall"},
		{Question{Kind: Select, Title: "Action"}, []string{"add"}, "> Action: add"},
		{Question{Kind: Input, Title: "Host"}, []string{"api.example.com"}, "> Host: api.example.com"},
		{Question{Kind: Input, Title: "Ports", Default: []string{"443"}}, []string{"443"}, "> Ports [443]:"},
		{Question{Kind: Input, Title: "Ports", Default: []string{"22,443"}}, []string{"22,443,9418"}, "> Ports [22,443]: 22,443,9418"},
		{Question{Kind: Select, Title: "Secret", Default: []string{"none"}}, []string{"none"}, "> Secret [none]:"},
		{Question{Kind: Select, Title: "Secret", Default: []string{"none"}}, []string{"EXAMPLE_TOKEN"}, "> Secret [none]: EXAMPLE_TOKEN"},
		{Question{Kind: Multi, Title: "Inject", Default: []string{"header"}}, []string{"header"}, "> Inject [header]:"},
		{Question{Kind: Multi, Title: "Inject", Default: []string{"header"}}, []string{"header", "body"}, "> Inject [header]: header, body"},
		{Question{Kind: Confirm, Title: "Apply?"}, []string{"yes"}, "> Apply? [y/N] y"},
		{Question{Kind: Confirm, Title: "Create an o3s home in /home/ubuntu/o3s?"}, []string{"no"}, "> Create an o3s home in /home/ubuntu/o3s? [y/N] n"},
		{Question{Kind: Hidden, Title: "Value for EXAMPLE_TOKEN (empty to skip)"}, []string{"s3cr3t"}, "> Value for EXAMPLE_TOKEN (empty to skip): ****"},
	}
	// Check every answered question
	for _, tt := range tests {
		// Report a line that differs
		if got := collapse(tt.q, tt.answer); got != tt.want {
			t.Errorf("collapse(%q, %q) = %q, want %q", tt.q.Title, tt.answer, got, tt.want)
		}
	}
}
