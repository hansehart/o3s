package usagetest

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// parameters reads the parameters of the PICT model and their values, a failure's ~ dropped.
func parameters(t *testing.T) map[string][]string {
	t.Helper()
	model, err := os.ReadFile("testdata/setups.pict")
	// Stop where the model stays unread
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	out := map[string][]string{}
	// Read every line that defines a parameter
	for line := range strings.SplitSeq(string(model), "\n") {
		name, values, ok := strings.Cut(line, ":")
		// Leave out comments, constraints and blank lines
		if !ok || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "IF ") {
			continue
		}
		// Take every value of the parameter
		for v := range strings.SplitSeq(values, ",") {
			out[name] = append(out[name], strings.TrimPrefix(strings.TrimSpace(v), "~"))
		}
	}
	return out
}

// values names a setup's value of every parameter.
func values(s Setup) map[string]string {
	return map[string]string{"Session": s.Session, "Agent": s.Agent, "Size": fmt.Sprintf("%dx%d", s.Cols, s.Rows), "Fault": s.Fault}
}

// The table holds every two values of the model together in some setup, outside the pair the model rules out: an
// agent that goes away from a session that held none.
func TestSetupsCoverEveryPair(t *testing.T) {
	setups, err := Setups()
	// Stop where the table stays unread
	if err != nil {
		t.Fatalf("Setups() = %v", err)
	}
	params := parameters(t)
	seen := map[string]bool{}
	// Note every pair of values a setup holds
	for _, s := range setups {
		v := values(s)
		// Pair every parameter with every other
		for a := range v {
			// Pair it with every other parameter
			for b := range v {
				seen[a+"="+v[a]+" "+b+"="+v[b]] = true
			}
		}
	}

	// Check every pair of values of two parameters
	for a, as := range params {
		// Pair with every other parameter
		for b, bs := range params {
			// Leave a parameter unpaired with itself
			if a == b {
				continue
			}
			// Check every value of the one with every value of the other
			for _, x := range as {
				// Check it with every value of the other
				for _, y := range bs {
					pair := a + "=" + x + " " + b + "=" + y
					ruledOut := a == "Fault" && x == "agent-gone" && b == "Agent" && y == "none" ||
						b == "Fault" && y == "agent-gone" && a == "Agent" && x == "none"
					// Report a pair the model allows that no setup holds
					if !ruledOut && !seen[pair] {
						t.Errorf("no setup holds %s", pair)
					}
				}
			}
		}
	}
}
