package repo

// Which parts of a repository move together.
//
// The same reading as cochange.go and a different question. That one asks
// "this file changed — what usually comes with it and did not", which is a
// warning about one change. This asks "across the history, which of these
// two ever move in the same commit", which is a fact about the repository
// and true whether anything changed today or not.
//
// It is what lets a map put things that belong together next to each other.
// A drawing whose cells touch is a drawing that claims adjacency means
// something; ordered by name it means the alphabet, and the alphabet is not
// a fact about the code.

import (
	"fmt"
	"strings"
)

// Near is how often two paths were committed together.
type Near struct {
	// A and B are paths from the root of the repository, and they are
	// siblings: both directories under one parent, or both files in one.
	A, B  string
	Times int
}

// Neighbours is every pair of siblings the history has moved together, at
// every depth of the tree.
//
// One walk of the log answers for the whole tree. A file's ancestors move
// with it by definition — `internal/ui/x.go` changing is `internal/ui`
// changing is `internal` changing — so the same commit is counted once at
// each depth, between whichever ancestors are siblings there.
//
// Only siblings, because that is the only pair a map ever draws side by
// side: a level is the children of one directory, and `internal` is never
// on screen beside `internal/ui`.
func (r Repo) Neighbours(wtDir string) ([]Near, error) {
	// The worktree and not the repository: a task's history is the base
	// branch's, and this is read from the checkout the task is in for the
	// same reason the diff is.
	out, err := git(wtDir, "log", "--name-only",
		fmt.Sprintf("-n%d", commitsRead), "--pretty=format:%x00%H")
	if err != nil {
		return nil, fmt.Errorf("read the history of %s: %w", r.Name, err)
	}

	return pairsIn(commitsOf(out)), nil
}

// pairsIn is the counting, over commits already read.
//
// Apart from the reading above so that it can be tested without a
// repository: what these rules do to a history is the part worth holding
// still, and a test that needs five hundred real commits to ask about one
// of them is a test nobody writes.
func pairsIn(commits [][]string) []Near {
	pairs := map[string]map[string]int{}

	for _, files := range commits {
		// A commit that touched fifty files says nothing about which of
		// them belong together — a squashed merge, a formatting pass, a
		// directory renamed. cochange.go draws the line in the same place
		// and for the same reason.
		if len(files) > crowdedCommit {
			continue
		}

		tally(pairs, files)
	}

	return listed(pairs)
}

// tally adds one commit's pairs, at every depth its files reach.
func tally(pairs map[string]map[string]int, files []string) {
	// Once per depth, and the set is what stops a commit that touched six
	// files under one directory from counting that directory six times
	// against its neighbour.
	deep := 0
	for _, f := range files {
		deep = max(deep, strings.Count(f, "/")+1)
	}

	for at := 1; at <= deep; at++ {
		seen := map[string]bool{}

		var here []string

		for _, f := range files {
			cut := ancestor(f, at)
			if cut == "" || seen[cut] {
				continue
			}

			seen[cut] = true

			here = append(here, cut)
		}

		for i, a := range here {
			for _, b := range here[i+1:] {
				// Siblings only: two paths at the same depth under the same
				// parent. Everything else at this depth is a pair no level
				// of the map ever puts side by side.
				if parent(a) != parent(b) {
					continue
				}

				add(pairs, a, b)
				add(pairs, b, a)
			}
		}
	}
}

// ancestor is the first n segments of a path, and empty where it has fewer.
func ancestor(path string, n int) string {
	segments := strings.Split(path, "/")
	if len(segments) < n {
		return ""
	}

	return strings.Join(segments[:n], "/")
}

// parent is everything above a path, and empty for one at the root.
func parent(path string) string {
	at := strings.LastIndex(path, "/")
	if at < 0 {
		return ""
	}

	return path[:at]
}

// add counts one ordered pair.
func add(pairs map[string]map[string]int, a, b string) {
	if pairs[a] == nil {
		pairs[a] = map[string]int{}
	}

	pairs[a][b]++
}

// listed is the pairs worth carrying, each once.
//
// A pair seen twice is two coincidences; floorTimes is where cochange.go
// stops calling that a pattern, and a map drawn from coincidences is a map
// that rearranges itself every time somebody commits.
func listed(pairs map[string]map[string]int) []Near {
	var out []Near

	for a, with := range pairs {
		for b, times := range with {
			// Once per pair rather than twice: the counting above is
			// symmetric, and a caller reading both directions would weigh
			// every neighbour double.
			if a < b && times >= floorTimes {
				out = append(out, Near{A: a, B: b, Times: times})
			}
		}
	}

	return out
}
