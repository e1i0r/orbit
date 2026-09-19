package repo

// The readings of git's own output, against bytes nothing here wrote.
//
// Every one of these is handed whatever `git` printed. The shape is git's to
// change, a path may hold anything a filesystem allows, and a repository
// somebody has been working in for ten years holds commit messages in every
// language there is. None of these readings may fall over on any of it: a
// run that cannot read a diff is a run that cannot say what it changed.

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzNumstat reads git's three columns.
//
// Two claims. A line it cannot read is skipped rather than guessed at — so
// what comes back is only ever lines that had all three columns and a path.
// And the two figures are either a count or the -1 that says git had none,
// never some other negative that a budget would subtract.
func FuzzNumstat(f *testing.F) {
	for _, seed := range []string{
		"",
		"3\t1\tinternal/db/pr.go",
		"-\t-\tlogo.png",
		"3\t1\tinternal/db/pr.go\n0\t0\tempty.txt",
		"3\t1\t",
		"3\t1",
		"not\ta\tnumber.go",
		"3\t1\tpath with spaces.go",
		"3\t1\tpath\twith\ttabs.go",
		"\t\t",
		"\n\n\n",
		"3\t1\t\x00binary.go",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, out string) {
		for _, c := range numstat(out) {
			if c.Path == "" {
				t.Errorf("numstat(%q) answered a change with no path: %+v", out, c)
			}

			if c.Added < -1 || c.Deleted < -1 {
				t.Errorf("numstat(%q) answered %+v, and neither figure may be past -1", out, c)
			}

			// Binary is the one reading of a negative, and Lines is what a
			// diff budget adds up: a file git counted no lines for must not
			// take lines off the total.
			if c.Binary() && c.Lines() != 0 {
				t.Errorf("numstat(%q) answered a binary file weighing %d: %+v", out, c.Lines(), c)
			}

			if !c.Binary() && c.Lines() < 0 {
				t.Errorf("numstat(%q) answered a text file weighing %d: %+v", out, c.Lines(), c)
			}
		}
	})
}

// FuzzCommitsOf reads the history one commit at a time.
//
// The first line of each block is the commit's own hash and is not a file.
// Counted as one it put the crowded-commit cutoff a file out and left a key
// nothing matches in the pair map of every commit read — which is why what
// comes back may hold no empty path and no commit with nothing in it.
func FuzzCommitsOf(f *testing.F) {
	for _, seed := range []string{
		"",
		"\x00abc123\ninternal/db/pr.go",
		"\x00abc123\na.go\nb.go\x00def456\nc.go",
		"\x00abc123",
		"\x00",
		"\x00\x00\x00",
		"no separator at all\nsecond line",
		"\x00abc123\n\n\n  \n a.go ",
		"\x00abc123\nel árbol.go",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, out string) {
		for i, commit := range commitsOf(out) {
			if len(commit) == 0 {
				t.Errorf("commitsOf(%q) answered commit %d with nothing in it", out, i)
			}

			for _, path := range commit {
				if path == "" {
					t.Errorf("commitsOf(%q) answered an empty path in commit %d", out, i)
				}

				if strings.TrimSpace(path) != path {
					t.Errorf("commitsOf(%q) answered %q, which still has its whitespace", out, path)
				}
			}
		}
	})
}

// FuzzSplitLines is what every other reading here is built on: git's output
// as the lines that say something.
func FuzzSplitLines(f *testing.F) {
	for _, seed := range []string{
		"", "\n", "\n\n\n", "  \t  ", "one\ntwo", " one \n\n two ",
		"\r\none\r\n", "línea\n", "\x00\n",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, out string) {
		lines := splitLines(out)
		for i, line := range lines {
			if line == "" {
				t.Errorf("splitLines(%q) answered an empty line at %d", out, i)
			}

			if strings.TrimSpace(line) != line {
				t.Errorf("splitLines(%q) answered %q, which still has its whitespace", out, line)
			}

			if strings.Contains(line, "\n") {
				t.Errorf("splitLines(%q) answered %q, which is more than one line", out, line)
			}
		}

		// Nothing is invented: what comes back was in what went in. Valid
		// text only, because every caller puts these somewhere a person or
		// a column will read them.
		if utf8.ValidString(out) {
			for _, line := range lines {
				if !strings.Contains(out, line) {
					t.Errorf("splitLines(%q) answered %q, which is not in it", out, line)
				}
			}
		}
	})
}
