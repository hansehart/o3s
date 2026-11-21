package usagetest

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// The orders hold every event once each, and every three events in every one of their six orders.
func TestOrdersCoverEveryTriple(t *testing.T) {
	// Check every count of events up to more than the suite runs
	for n := 3; n <= 14; n++ {
		orders := Orders(n)
		open := triples(n)
		// Close every triple an order holds, checking it holds every event once
		for _, o := range orders {
			// Report an order that misses or repeats an event
			if got := slices.Sorted(slices.Values(o)); !slices.Equal(got, Orders(n)[0]) {
				t.Fatalf("Orders(%d) holds %v, want every event once", n, o)
			}
			cover(open, o)
		}

		// Report a triple no order holds
		if len(open) > 0 {
			t.Errorf("Orders(%d) leaves %d triples open", n, len(open))
		}
	}
}

// The same count of events gives the same orders on every run.
func TestOrdersRepeat(t *testing.T) {
	first, again := Orders(10), Orders(10)

	// Report orders that differ between runs
	if diff := cmp.Diff(first, again); diff != "" {
		t.Errorf("Orders(10) differs between runs (-first +again):\n%s", diff)
	}
}

// An order holds every three of its events in the order they happen in it.
func TestCoveredTriples(t *testing.T) {
	got := covered([]int{2, 0, 1, 3})

	// Report other triples than the order's
	if diff := cmp.Diff([]triple{{2, 0, 1}, {2, 0, 3}, {2, 1, 3}, {0, 1, 3}}, got); diff != "" {
		t.Errorf("covered() (-want +got):\n%s", diff)
	}
}
