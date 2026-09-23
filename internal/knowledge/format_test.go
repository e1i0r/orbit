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

// TestALanguageScopeWithNoLanguageTakesItFromWhereTheFileIs. The rules of one
// language live in a directory named after it, so a file that says it is
// about a language without saying which is answered by where it was found —
// and one that says which keeps what it said.
func TestALanguageScopeWithNoLanguageTakesItFromWhereTheFileIs(t *testing.T) {
	// The file the reading came out of: its own directory is the language.
	const from = "lang/go/amounts-are-cents.md"

	got := scopeFrom(map[string]string{keyScope: "language"}, from, "")
	if got.Lang != "go" {
		t.Errorf("a rule under %q is about %q, want go", from, got.Lang)
	}

	said := scopeFrom(map[string]string{keyScope: "language", keyLang: "rust"}, from, "")
	if said.Lang != "rust" {
		t.Errorf("a rule that says it is about rust was read as %q", said.Lang)
	}

	// And a rule about a language is about no checkout, wherever its file
	// happens to sit.
	if got.Repo != "" {
		t.Errorf("a rule about a language is filed under the checkout %q", got.Repo)
	}
}

// TestTheLastPartOfAPathIsTheWholeOfItWhenThereIsOnlyOne, which is what a
// language directory at the top of a state root looks like.
func TestTheLastPartOfAPathIsTheWholeOfItWhenThereIsOnlyOne(t *testing.T) {
	cases := map[string]string{
		"go":              "go",
		"lang/go":         "go",
		"a/b/c/rust":      "rust",
		"/leading/python": "python",
		"":                "",
	}

	for path, want := range cases {
		if got := lastSegment(path); got != want {
			t.Errorf("the last part of %q is %q, want %q", path, got, want)
		}
	}
}

// TestARuleCannotDriveTheTerminal. A rule is a file, and a supervising
// model writes most of them: the sentence is drawn on the knowledge screen
// and handed to the next engine. One holding ESC[2J cleared the reader's
// screen every time the screen was drawn.
func TestARuleCannotDriveTheTerminal(t *testing.T) {
	body := "---\n" +
		"id: r1\n" +
		"source: human\n" +
		"why: because \x1b[31mred\x1b[0m\n" +
		"ref: see \x1b[2J\n" +
		"---\n" +
		"never \x1b]0;stolen\x07do that\a\n"

	r, err := decode(body, "repo/orbit/r1.md", "/w/orbit")
	if err != nil {
		t.Fatalf("decode the rule: %v", err)
	}

	said := r.Phrase + r.Why + r.Ref + r.Check
	if strings.ContainsAny(said, "\x1b\x07") {
		t.Errorf("a rule read from a file carries control characters: %q", said)
	}

	for _, want := range []string{"never do that", "because red", "see"} {
		if !strings.Contains(said, want) {
			t.Errorf("taming the rule lost %q: %q", want, said)
		}
	}
}
