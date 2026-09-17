package knowledge

// How a rule written by hand is read back: the kind its header names, and
// where the file sits when the header says nothing.

import (
	"strings"
	"testing"
)

// TestAFactWithNoKindInItsHeaderIsPlacedByWhereItSits.
//
// The store is files in directories, and a file written by hand — or by an
// older Orbit — carries no kind. Where it is is the answer: under lang/ it
// is about a language, at the top of a repository's own directory it is
// about that repository, and deeper in one it is about that directory.
func TestAFactWithNoKindInItsHeaderIsPlacedByWhereItSits(t *testing.T) {
	for _, c := range []struct {
		what  string
		name  string
		where string
		repo  string
		want  Kind
	}{
		{"a kind said out loud", kindNames[Symbol], "anywhere", "orbit", Symbol},
		{"under lang", "", langDir + "/go/never-discard.md", "", Language},
		{"nowhere in particular", "", "prs-in-english.md", "", General},
		{"the top of a repository", "", "the-fuzz-tests-hang.md", "orbit", Repo},
		{"deeper in one", "", "internal/ui/measure-in-cells.md", "orbit", Dir},
	} {
		if got := kindNamed(c.name, c.where, c.repo); got != c.want {
			t.Errorf("%s: kindNamed(%q, %q, %q) = %v, want %v",
				c.what, c.name, c.where, c.repo, got, c.want)
		}
	}
}

// TestARuleWrittenByAnOlderOrbitStillReads. Orbit wrote a `used:` count into
// every rule for a while and never incremented it, so the key is on disk in
// files people have. Dropping the field must not turn those into rules that
// will not parse.
func TestARuleWrittenByAnOlderOrbitStillReads(t *testing.T) {
	was := "---\n" +
		"scope: general\n" +
		"source: human\n" +
		"at: 2026-09-01T00:00:00Z\n" +
		"used: 12\n" +
		"---\n" +
		"\n" +
		"never log a card number\n"

	f, err := decode(was, "", "")
	if err != nil {
		t.Fatalf("a rule an older orbit wrote did not read: %v", err)
	}

	if f.Phrase != "never log a card number" {
		t.Errorf("the sentence came back as %q", f.Phrase)
	}

	// And it is not written back out: the next save is the file without it.
	if strings.Contains(encode(f), "used:") {
		t.Errorf("the count was written again:\n%s", encode(f))
	}
}
