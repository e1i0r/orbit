package repo

// What a change reaches, beyond the files it touched.
//
// Two readings, and both are the history's rather than the code's: which
// files usually come along and did not this time, and what the tests among
// them say they guarantee. Neither runs anything, neither parses a language,
// and a repository Orbit could not build still answers.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Impact is that reading for one worktree.
type Impact struct {
	// Changed is what the work touched, and Commits how many commits the
	// history was read from — a repository with nine commits in it says so
	// rather than letting three coincidences look like a pattern.
	Changed []string
	Commits int
	// Coupled is what usually comes along and did not, strongest first.
	Coupled []Coupled
	// Contracts is what the tests among those files say they hold, by their
	// own names. A test called rejects_negative_amounts is a sentence
	// somebody wrote about the behaviour, and it is the cheapest
	// specification any repository has.
	Contracts []Contract
}

// Anything is whether the reading found something worth a reader's eye.
func (i Impact) Anything() bool { return len(i.Coupled) > 0 }

// Contract is one thing a test says it holds.
type Contract struct {
	File string
	Says string
}

// Impact reads what the work in this worktree reaches.
func (r Repo) Impact(wtDir string) (Impact, error) {
	changes, err := r.WorktreeChanges(wtDir)
	if err != nil {
		return Impact{}, err
	}

	changed := make([]string, 0, len(changes))
	for _, c := range changes {
		changed = append(changed, c.Path)
	}

	got := Impact{Changed: changed}
	if len(changed) == 0 {
		return got, nil
	}

	// %x00 before the hash, so one commit is one block however many blank
	// lines its message has: see commitsOf.
	out, err := git(r.Path, "log", "--name-only", fmt.Sprintf("-n%d", commitsRead), "--pretty=format:%x00%H")
	if err != nil {
		return got, fmt.Errorf("read the history of %s: %w", r.Name, err)
	}

	commits := commitsOf(out)
	got.Commits = len(commits)
	got.Coupled = coChanged(commits, changed)
	got.Contracts = r.contracts(got.Coupled)

	return got, nil
}

// contracts is what the tests that were left out say they hold.
//
// Only the ones the history warned about: a list of every test in the
// repository is a file listing, and what makes these worth reading is that
// something changed under them and they did not.
func (r Repo) contracts(coupled []Coupled) []Contract {
	var out []Contract

	for _, c := range coupled {
		if !looksLikeTests(c.File) {
			continue
		}

		body, err := os.ReadFile(filepath.Join(r.Path, c.File))
		if err != nil {
			// A file the history knows and the checkout does not is a file
			// somebody deleted. It is not a fault and it is not a contract.
			continue
		}

		for _, says := range testNames(string(body)) {
			out = append(out, Contract{File: c.File, Says: says})
		}
	}

	return out
}

// looksLikeTests is whether a path is a test file, by the three conventions
// that cover nearly every repository: a name that starts or ends with test,
// or one that says spec.
func looksLikeTests(path string) bool {
	name := strings.ToLower(filepath.Base(path))

	switch {
	case strings.HasPrefix(name, "test_"), strings.HasPrefix(name, "test."):
		return true
	case strings.Contains(name, "_test."), strings.Contains(name, ".test."):
		return true
	case strings.Contains(name, "_spec."), strings.Contains(name, ".spec."):
		return true
	}

	return strings.Contains(strings.ToLower(path), "/tests/")
}
