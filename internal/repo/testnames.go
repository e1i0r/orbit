package repo

// A test's name, read as the sentence somebody meant by it.
//
// Nothing here parses a language. It looks for the three shapes a test is
// declared in — Go's func TestX, Python's def test_x, and the it("...") of
// every JavaScript runner — and turns what it finds into words. A name is
// not a proof of anything, but it is the cheapest specification a repository
// has and the only one that is written in English.

import (
	"regexp"
	"strings"
)

// mostContracts is how many are read out of one file. A suite of two hundred
// is a file listing, and the point is to say what is at risk.
const mostContracts = 6

// declared is the three shapes, in one pass: Go, Python, and the string a
// JavaScript runner takes.
var declared = regexp.MustCompile(`(?m)^\s*(?:func (Test\w+)|def (test_\w+)|(?:it|test)\(\s*['"` + "`" + `]([^'"` + "`" + `]+))`)

// testNames is what the tests in a file say they hold, in the order they are
// written.
func testNames(body string) []string {
	var out []string

	for _, found := range declared.FindAllStringSubmatch(body, -1) {
		says := readable(firstOf(found[1], found[2], found[3]))
		if says == "" {
			continue
		}

		out = append(out, says)

		if len(out) == mostContracts {
			break
		}
	}

	return out
}

// firstOf is whichever of the three shapes matched.
func firstOf(all ...string) string {
	for _, s := range all {
		if s != "" {
			return s
		}
	}

	return ""
}

// readable turns a declaration into the sentence it was: the prefix off, the
// underscores and the humps out.
func readable(name string) string {
	name = strings.TrimPrefix(name, "Test")
	name = strings.TrimPrefix(name, "test_")
	name = strings.ReplaceAll(name, "_", " ")

	var b strings.Builder

	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteRune(' ')
		}

		b.WriteRune(r)
	}

	return strings.ToLower(strings.TrimSpace(b.String()))
}
