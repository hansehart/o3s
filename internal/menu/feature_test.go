package menu

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// fakeLookup is a store that offers the tool, whose feature needs two secrets still waiting for values.
type fakeLookup struct{}

// Features lists the ids the store offers, and fails for any other store.
func (fakeLookup) Features(store string) ([]string, error) {
	// Turn away a store out of reach
	if store != "registry.example.com/features" {
		return nil, errors.New("read the features of " + store + ": unreachable")
	}
	return []string{"other", "tool"}, nil
}

// Add shows what the tool brings.
func (fakeLookup) Add(id string) (Preview, error) {
	return Preview{Lines: []string{id + ":1", "  + example.com:443  EXAMPLE_TOKEN ⚠", "  + example.org:443  OTHER_TOKEN ⚠"}, Waiting: []string{"EXAMPLE_TOKEN", "OTHER_TOKEN"}}, nil
}

// Remove shows what leaves with the tool.
func (fakeLookup) Remove(pkg string) (Preview, error) {
	return Preview{Lines: []string{pkg, "  - example.com:443"}}, nil
}

// aHomeWithTheTool is aHome holding the tool and reading its store through the fake.
func aHomeWithTheTool() Home {
	h := aHome
	h.Features = []string{"registry.example.com/features/tool"}
	h.Lookup = fakeLookup{}
	return h
}

// The feature dialogs preview their change, ask for waiting values, and run the commands they lead to.
func TestFeatureDialogsEndInTheirCommands(t *testing.T) {
	value := "s3cr3t"
	tests := []struct {
		name    string
		answers [][]string
		titles  []string
		want    []Command
	}{
		{
			"adding a feature with one value given and one skipped",
			[][]string{{"feature"}, {"add"}, {"registry.example.com/features"}, {"tool"}, {"yes"}, {value}, {""}},
			[]string{"What do you want to do?", "Action", "Store", "Feature", "Apply?", "Value for EXAMPLE_TOKEN (empty to skip)", "Value for OTHER_TOKEN (empty to skip)"},
			[]Command{
				{Args: []string{"secret", "add", "EXAMPLE_TOKEN"}, Stdin: &value},
				{Args: []string{"feature", "add", "registry.example.com/features/tool"}},
			},
		},
		{
			"declining a feature",
			[][]string{{"feature"}, {"add"}, {"registry.example.com/features"}, {"tool"}, {"no"}},
			[]string{"What do you want to do?", "Action", "Store", "Feature", "Apply?"},
			nil,
		},
		{
			"a store out of reach",
			[][]string{{"feature"}, {"add"}, {"registry.example.org/features"}},
			[]string{"What do you want to do?", "Action", "Store"},
			[]Command{{Note: "⚠ read the features of registry.example.org/features: unreachable"}},
		},
		{
			"removing a feature",
			[][]string{{"feature"}, {"rm"}, {"registry.example.com/features/tool"}, {"yes"}},
			[]string{"What do you want to do?", "Action", "Feature", "Apply?"},
			[]Command{{Args: []string{"feature", "rm", "registry.example.com/features/tool"}}},
		},
		{
			"listing the features",
			[][]string{{"feature"}, {"ls"}},
			[]string{"What do you want to do?", "Action"},
			[]Command{{Args: []string{"feature", "ls"}}},
		},
	}
	h := aHomeWithTheTool()
	h.Stores = append(h.Stores, "registry.example.org/features")
	// Walk every dialog
	for _, tt := range tests {
		asked, got := walk(t, h, tt.answers...)

		// Report questions that differ
		if diff := cmp.Diff(tt.titles, titles(asked)); diff != "" {
			t.Errorf("%s: questions (-want +got):\n%s", tt.name, diff)
		}
		// Report commands that differ
		if diff := cmp.Diff(tt.want, got); diff != "" {
			t.Errorf("%s: commands (-want +got):\n%s", tt.name, diff)
		}
	}
}

// Apply? shows what the feature brings or takes, as the CLI prints it.
func TestFeatureApplyShowsThePreview(t *testing.T) {
	h := aHomeWithTheTool()
	added, _ := walk(t, h, []string{"feature"}, []string{"add"}, []string{"registry.example.com/features"}, []string{"tool"}, []string{"no"})
	removed, _ := walk(t, h, []string{"feature"}, []string{"rm"}, []string{"registry.example.com/features/tool"}, []string{"no"})

	got := [][]string{added[4].Preview, removed[3].Preview}

	want := [][]string{
		{"registry.example.com/features/tool:1", "  + example.com:443  EXAMPLE_TOKEN ⚠", "  + example.org:443  OTHER_TOKEN ⚠"},
		{"registry.example.com/features/tool", "  - example.com:443"},
	}
	// Report previews that differ
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("previews (-want +got):\n%s", diff)
	}
}

// A feature dialog with nothing to choose from ends with a note.
func TestFeatureDialogsWithNothingToChooseEndWithANote(t *testing.T) {
	h := Home{Dir: "/home/ubuntu/o3s", Lookup: fakeLookup{}}
	_, add := walk(t, h, []string{"feature"}, []string{"add"})
	_, rm := walk(t, h, []string{"feature"}, []string{"rm"})

	got := [][]Command{add, rm}

	want := [][]Command{{{Note: "No store to choose from"}}, {{Note: "No feature to remove"}}}
	// Report notes that differ
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("notes (-want +got):\n%s", diff)
	}
}

// The first question offers the features between the stores and the update.
func TestTheFirstQuestionOffersFeatures(t *testing.T) {
	q, _ := next(aHome, nil)

	got := q.Options

	want := []Option{{Value: "init"}, {Value: "firewall"}, {Value: "secret"}, {Value: "store"}, {Value: "feature"}, {Value: "update"}}
	// Report options that differ
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("options (-want +got):\n%s", diff)
	}
}
