package task

// Which rules a prompt carries when Orbit knows more than one can hold.

import (
	"slices"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// atMostTold is how many rules one prompt carries.
//
// A number and not "all of them", because this is one of the few things in
// Orbit that grows on its own: it learns four ways and keeps everything it
// is told, and nobody prunes. Without a ceiling every phase of every task
// pays for the whole store in tokens before it reads a line of code, and the
// rules at the end of a long list are the ones a model with a full context
// stops looking at — silently, so the first sign of it is work that broke a
// rule nobody can see was dropped.
//
// Forty because it is more than any checkout has today and small enough that
// a phase reads all of it. The number matters far less than that there is
// one and that what it cuts is decided rather than incidental.
const atMostTold = 40

// told is the rules one prompt carries, and how many were left out.
//
// Two orders, and they are not the same order.
//
// What to keep is decided narrowest first, because a rule about the file
// being worked in is worth more to this phase than a rule about every
// project on the machine — and before that, the rules that stop the work,
// because those are the ones that send it back. A phase that never saw a
// gate walks into it.
//
// What is written is ordered widest first, which is the order the store
// already hands them over in and the reason it does: the agent reads them
// in order, so what was written about the narrowest place is the last thing
// it reads and has the last word. Truncating the list Orbit already ordered
// would have cut exactly the rules worth keeping.
func told(facts []knowledge.Rule) (kept []knowledge.Rule, left int) {
	if len(facts) <= atMostTold {
		return facts, 0
	}

	byWorth := slices.Clone(facts)
	slices.SortStableFunc(byWorth, func(a, b knowledge.Rule) int {
		if stops(a) != stops(b) {
			if stops(a) {
				return -1
			}

			return 1
		}

		return b.Scope.Depth() - a.Scope.Depth()
	})

	kept = byWorth[:atMostTold]
	slices.SortStableFunc(kept, func(a, b knowledge.Rule) int {
		return a.Scope.Depth() - b.Scope.Depth()
	})

	return kept, len(facts) - atMostTold
}

// stops is whether a rule is enforced by a gate rather than merely advised.
func stops(f knowledge.Rule) bool { return f.Action() == knowledge.Stops }
