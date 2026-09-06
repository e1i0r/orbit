package repo

// What else usually changes when this file changes.
//
// A diff says what the work touched. It cannot say what the work forgot —
// and the thing that goes wrong at three in the morning is almost never the
// file somebody edited, it is the one that always went with it and did not
// this time. The history knows: two files committed together thirty-six
// times out of thirty-nine are one change wearing two names, whatever the
// import graph says about them.
//
// This reads the history and nothing else. No language, no parser, no build:
// a repository of Python and a repository of Go answer the same way, and a
// repository whose imports Orbit could never resolve answers too.

import (
	"sort"
	"strings"
)

// The shape of the reading.
const (
	// commitsRead is how far back the history is walked. Far enough for a
	// pattern to show, and short enough that a repository's first year does
	// not outvote what it has become.
	commitsRead = 500
	// floorTimes is how many commits two files have to have shared before
	// the pair is worth saying out loud. Two out of two is a hundred
	// percent and means nothing.
	floorTimes = 4
	// floorRatio is how often the other file has to follow before it is
	// worth a warning.
	floorRatio = 0.5
	// crowdedCommit is a commit that touched so many files that it says
	// nothing about which of them belong together: a squashed merge, a
	// formatting pass, a rename of a directory.
	crowdedCommit = 50
	// mostCoupled is how many are shown. The list is a warning, not a
	// report, and past a handful nobody reads it.
	mostCoupled = 8
)

// Coupled is one file the history says follows another.
type Coupled struct {
	// File is the one that usually comes along, and With is the changed
	// file it follows.
	File string
	With string
	// Times is how many of the commits that touched With touched File too,
	// and Of is how many touched With at all.
	Times int
	Of    int
}

// Ratio is how often it follows, between 0 and 1.
func (c Coupled) Ratio() float64 {
	if c.Of == 0 {
		return 0
	}

	return float64(c.Times) / float64(c.Of)
}

// coChanged is every file the history says follows one of the changed ones
// and was not changed this time, strongest first.
//
// A file that was changed is not reported against itself or against its
// neighbours: the whole question is what was left out.
func coChanged(commits [][]string, changed []string) []Coupled {
	var (
		touched = set(changed)
		with    = map[string]int{}            // how many commits touched each changed file
		pairs   = map[string]map[string]int{} // changed file → other file → how many together
	)

	for _, files := range commits {
		if len(files) > crowdedCommit {
			continue
		}

		for _, one := range files {
			if !touched[one] {
				continue
			}

			with[one]++

			for _, other := range files {
				if other == one || touched[other] {
					continue
				}

				if pairs[one] == nil {
					pairs[one] = map[string]int{}
				}

				pairs[one][other]++
			}
		}
	}

	return strongest(pairs, with)
}

// strongest keeps the pairs worth a warning, the strongest of each file, and
// the handful of those a reader will actually look at.
func strongest(pairs map[string]map[string]int, with map[string]int) []Coupled {
	best := map[string]Coupled{}

	for one, others := range pairs {
		for other, times := range others {
			got := Coupled{File: other, With: one, Times: times, Of: with[one]}
			if got.Times < floorTimes || got.Ratio() < floorRatio {
				continue
			}

			// One file can follow two of the changed ones. It is said once,
			// against the one it follows most closely.
			if held, seen := best[other]; seen && held.Ratio() >= got.Ratio() {
				continue
			}

			best[other] = got
		}
	}

	out := make([]Coupled, 0, len(best))
	for _, c := range best {
		out = append(out, c)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Ratio() != out[j].Ratio() {
			return out[i].Ratio() > out[j].Ratio()
		}

		return out[i].Times > out[j].Times
	})

	if len(out) > mostCoupled {
		out = out[:mostCoupled]
	}

	return out
}

// set is the paths, for asking whether one of them was touched.
func set(paths []string) map[string]bool {
	out := make(map[string]bool, len(paths))
	for _, p := range paths {
		out[p] = true
	}

	return out
}

// commitsOf reads `git log --name-only` into one list of paths per commit.
//
// The separator is a NUL rather than a blank line: a commit message has
// blank lines in it, and a paragraph of one read as a list of files is how a
// sentence ends up warning somebody about a file called "Fixes:".
func commitsOf(out string) [][]string {
	var commits [][]string

	for _, block := range strings.Split(out, "\x00") {
		var files []string

		for _, line := range strings.Split(block, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				files = append(files, line)
			}
		}

		if len(files) > 0 {
			commits = append(commits, files)
		}
	}

	return commits
}
