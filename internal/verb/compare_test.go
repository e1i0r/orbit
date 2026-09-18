package verb

// The comparison in the shape a surface draws it.

import (
	"errors"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
)

// TestWhatBrokeAndWhatWasFixedAreClaimedOnlyWhereBothSidesRan.
//
// A check that could not run on one side has no verdict to compare, and
// calling that a regression is how a reader learns to distrust the whole
// section. The two are read off exit codes, so each of the four ways two
// exit codes can sit is its own answer.
func TestWhatBrokeAndWhatWasFixedAreClaimedOnlyWhereBothSidesRan(t *testing.T) {
	gone := errors.New("no such directory")

	for _, one := range []struct {
		why   string
		base  repo.Ran
		now   repo.Ran
		broke bool
		fixed bool
	}{
		{
			"it passed before the change and fails after it",
			repo.Ran{Exit: 0},
			repo.Ran{Exit: 1},
			true, false,
		},
		{
			"it failed before the change and passes after it",
			repo.Ran{Exit: 1},
			repo.Ran{Exit: 0},
			false, true,
		},
		{"it passed on both sides", repo.Ran{Exit: 0}, repo.Ran{Exit: 0}, false, false},
		{"it failed on both sides", repo.Ran{Exit: 1}, repo.Ran{Exit: 1}, false, false},
		{
			"the base side never ran, whatever the other one says",
			repo.Ran{Failed: gone},
			repo.Ran{Exit: 1},
			false, false,
		},
		{
			"and the changed side never ran",
			repo.Ran{Exit: 1},
			repo.Ran{Failed: gone},
			false, false,
		},
		{"neither side ran", repo.Ran{Failed: gone}, repo.Ran{Failed: gone}, false, false},
	} {
		got := bothSides([]repo.Divergence{{
			Check: repo.Check{Name: "test", Command: "go test ./..."},
			Base:  one.base,
			Now:   one.now,
		}})

		if len(got) != 1 {
			t.Fatalf("one check came across as %d rows", len(got))
		}

		if got[0].Broke != one.broke || got[0].Fixed != one.fixed {
			t.Errorf("broke %v, fixed %v when %s — want %v and %v",
				got[0].Broke, got[0].Fixed, one.why, one.broke, one.fixed)
		}
	}
}
