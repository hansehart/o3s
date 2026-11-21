// Package usagetest plans how the usage suite uses o3s: the orders its commands run in and the setups it runs them on.
package usagetest

import (
	"math/rand/v2"
	"slices"
)

// trials is how many candidate orders each step weighs.
const trials = 1000

// Orders are orders of n events, each event once in every order, such that every three events appear in every one of
// their six orders across them: a sequence covering array of strength 3. It starts with the events in order and
// reversed, then adds the candidate that covers the most triples still open, and its reverse from six events on, until
// every triple is covered. A fixed seed draws the candidates, so the orders are the same on every run.
func Orders(n int) [][]int {
	first := make([]int, n)
	// Start with the events in order
	for i := range first {
		first[i] = i
	}
	last := slices.Clone(first)
	slices.Reverse(last)
	orders := [][]int{first, last}
	open := triples(n)
	cover(open, first)
	cover(open, last)
	rng := rand.New(rand.NewPCG(uint64(n), uint64(n))) //nolint:gosec // a fixed seed keeps the orders the same on every run
	// Add the order that covers the most open triples until none is open
	for len(open) > 0 {
		best, most := []int(nil), -1
		// Weigh every candidate the seed draws
		for range trials {
			candidate := rng.Perm(n)
			// Keep the candidate that covers the most
			if c := covers(open, candidate); c > most {
				best, most = candidate, c
			}
		}
		orders = append(orders, best)
		cover(open, best)
		// Add the reverse too from six events on, which covers the triples the order holds backwards
		if n > 5 {
			back := slices.Clone(best)
			slices.Reverse(back)
			orders = append(orders, back)
			cover(open, back)
		}
	}
	return orders
}

// triple is three distinct events in the order they happen.
type triple [3]int

// triples are every three distinct events of n in every order.
func triples(n int) map[triple]bool {
	out := map[triple]bool{}
	// Take every event first, second and third
	for i := range n {
		// Take every second event apart from the first
		for j := range n {
			// Take every third event apart from the other two
			for k := range n {
				// Keep three distinct events
				if i != j && i != k && j != k {
					out[triple{i, j, k}] = true
				}
			}
		}
	}
	return out
}

// covered are the triples an order holds, each three events in the order they happen in it.
func covered(order []int) []triple {
	var out []triple
	// Take every three positions in the order
	for i := 0; i < len(order)-2; i++ {
		// Take every later second position
		for j := i + 1; j < len(order)-1; j++ {
			// Take every later third position
			for k := j + 1; k < len(order); k++ {
				out = append(out, triple{order[i], order[j], order[k]})
			}
		}
	}
	return out
}

// covers counts the open triples an order holds.
func covers(open map[triple]bool, order []int) int {
	n := 0
	// Count every triple of the order still open
	for _, t := range covered(order) {
		// Count a triple still open
		if open[t] {
			n++
		}
	}
	return n
}

// cover closes the triples an order holds.
func cover(open map[triple]bool, order []int) {
	// Close every triple of the order
	for _, t := range covered(order) {
		delete(open, t)
	}
}
