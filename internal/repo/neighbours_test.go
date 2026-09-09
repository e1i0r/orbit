package repo

import (
	"fmt"
	"testing"
)

// TestOnlySiblingsAreNeighbours. A level of a map is the children of one
// directory, so `internal` is never drawn beside `internal/ui` — and a pair
// no drawing can put side by side is a pair worth nothing to carry.
func TestOnlySiblingsAreNeighbours(t *testing.T) {
	pairs := map[string]map[string]int{}
	for range floorTimes {
		tally(pairs, []string{"internal/ui/a.go", "internal/task/b.go", "site/index.html"})
	}

	for _, one := range listed(pairs) {
		if parent(one.A) != parent(one.B) {
			t.Errorf("%q and %q are not siblings and were counted together", one.A, one.B)
		}
	}
}

// TestTheSameCommitCountsOnceAtEachDepth. Six files under one directory are
// that directory changing once, not six times: counted per file, a busy
// package would out-weigh every pair in the repository by arithmetic.
func TestTheSameCommitCountsOnceAtEachDepth(t *testing.T) {
	pairs := map[string]map[string]int{}
	tally(pairs, []string{
		"internal/ui/a.go", "internal/ui/b.go", "internal/ui/c.go",
		"internal/task/x.go",
	})

	if got := pairs["internal/ui"]["internal/task"]; got != 1 {
		t.Errorf("one commit counted internal/ui with internal/task %d times, want 1", got)
	}
}

// TestACoincidenceIsNotAPattern. Two files seen together twice is two
// coincidences, and a map that seated cells by coincidences would rearrange
// itself on every commit.
func TestACoincidenceIsNotAPattern(t *testing.T) {
	pairs := map[string]map[string]int{}
	for range floorTimes - 1 {
		tally(pairs, []string{"a.go", "b.go"})
	}

	if got := listed(pairs); len(got) != 0 {
		t.Errorf("a pair seen %d times was carried: %v", floorTimes-1, got)
	}

	tally(pairs, []string{"a.go", "b.go"})

	if got := listed(pairs); len(got) != 1 {
		t.Errorf("a pair seen %d times was not carried: %v", floorTimes, got)
	}
}

// TestAPairIsCarriedOnce. The counting is symmetric, and a caller reading
// both directions would weigh every neighbour double.
func TestAPairIsCarriedOnce(t *testing.T) {
	pairs := map[string]map[string]int{}
	for range floorTimes {
		tally(pairs, []string{"a.go", "b.go"})
	}

	if got := listed(pairs); len(got) != 1 {
		t.Errorf("one pair came back as %d entries: %v", len(got), got)
	}
}

// TestACrowdedCommitSaysNothing. A squashed merge or a formatting pass
// touches everything and means nothing about what belongs together — so a
// pair that only ever appeared in commits like that is not a pair.
func TestACrowdedCommitSaysNothing(t *testing.T) {
	wide := make([]string, 0, crowdedCommit+1)
	for i := range crowdedCommit + 1 {
		wide = append(wide, fmt.Sprintf("f%d.go", i))
	}

	var crowd [][]string
	for range floorTimes {
		crowd = append(crowd, wide)
	}

	if got := pairsIn(crowd); len(got) != 0 {
		t.Errorf("a commit of %d files was counted: %v", len(wide), got[:1])
	}

	// And the same pair, in commits small enough to mean something.
	var small [][]string
	for range floorTimes {
		small = append(small, []string{"f0.go", "f1.go"})
	}

	if got := pairsIn(small); len(got) != 1 {
		t.Errorf("the same pair in small commits came back as %d entries", len(got))
	}
}
