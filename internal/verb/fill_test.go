package verb

// Reading the rest of a line into the fields a verb takes.
//
// It read zero, and it is the grammar every way in shares: a reader who
// learned `orbit note abc it needs a test` at a terminal types the same
// thing into a chat and means it. A second copy of these rules would be a
// second grammar, so there is one — and now something holds it.

import (
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// takes is a verb that takes those fields, in that order.
func takes(fields ...Field) Verb {
	return Verb{Name: "note", Takes: fields}
}

// aName is a field a reader names by one word.
func aName(name string, needed bool) Field {
	return Field{Name: name, Kind: Named, Needed: needed, About: nothingSaid}
}

// sentence is the field that takes everything left over.
func sentence(name string) Field {
	return Field{Name: name, Kind: Words, Needed: true, About: nothingSaid}
}

// nothingSaid stands in for the line a field explains itself with, which is
// not what any of these are about.
func nothingSaid(*words.Printer) string { return "" }

// TestTheOrderIsTheOrderTheVerbDeclares. One word per field it must have,
// and the field of words takes everything that is left — so the sentence at
// the end of `orbit note abc it needs a test` arrives whole.
func TestTheOrderIsTheOrderTheVerbDeclares(t *testing.T) {
	args := map[string]string{}

	left := Fill(
		takes(aName("id", true), sentence("text")),
		args,
		strings.Fields("ACME-1 it needs a test"),
	)

	if args["id"] != "ACME-1" {
		t.Errorf("the id arrived as %q", args["id"])
	}

	if args["text"] != "it needs a test" {
		t.Errorf("the sentence arrived as %q, want it whole", args["text"])
	}

	if len(left) != 0 {
		t.Errorf("%v had nowhere to go", left)
	}
}

// TestAFlagStillWins. A script that spelled it out meant it, and a
// positional word must not overwrite what was named.
func TestAFlagStillWins(t *testing.T) {
	args := map[string]string{"id": "ACME-9"}

	Fill(takes(aName("id", true), sentence("text")), args, strings.Fields("ACME-1 it needs a test"))

	if args["id"] != "ACME-9" {
		t.Errorf("the flag was overwritten with %q", args["id"])
	}

	// And the word that would have filled it is still the first word of the
	// sentence, because the field it was for was already answered.
	if args["text"] != "ACME-1 it needs a test" {
		t.Errorf("the sentence arrived as %q", args["text"])
	}
}

// TestALeadingDashDashIsTheShellsAndNotAWord. It is how a shell says the
// flags are over, and flag stops at the first positional anyway — so a
// caller that puts it after the id is saying so once the flags are over
// already. Kept, it became the first word of every note the window wrote.
func TestALeadingDashDashIsTheShellsAndNotAWord(t *testing.T) {
	args := map[string]string{}

	Fill(takes(aName("id", true), sentence("text")), args, []string{"--", "ACME-1", "it needs a test"})

	if args["id"] != "ACME-1" {
		t.Errorf("the id arrived as %q", args["id"])
	}

	if strings.HasPrefix(args["text"], "--") {
		t.Errorf("the shell's separator became part of the note: %q", args["text"])
	}
}

// TestAnOptionalNameIsStillANameWhenItIsThere. `reconcile PAY-1` narrows the
// sweep to one task, and a reader who typed it named something rather than
// adding noise.
func TestAnOptionalNameIsStillANameWhenItIsThere(t *testing.T) {
	args := map[string]string{}

	left := Fill(takes(aName("task", false)), args, []string{"PAY-1"})

	if args["task"] != "PAY-1" {
		t.Errorf("the optional name arrived as %q", args["task"])
	}

	if len(left) != 0 {
		t.Errorf("%v had nowhere to go", left)
	}
}

// TestAYesOrNoStaysOnItsFlag. A bare "true" on the line is a typo until
// proven otherwise, so the word is left over rather than switching
// something on.
func TestAYesOrNoStaysOnItsFlag(t *testing.T) {
	args := map[string]string{}

	yesOrNo := Field{Name: "restart", Kind: YesOrNo, About: nothingSaid}

	left := Fill(takes(yesOrNo), args, []string{"true"})

	if args["restart"] != "" {
		t.Errorf("a bare word switched %q on", args["restart"])
	}

	if !slices.Equal(left, []string{"true"}) {
		t.Errorf("it was left with %v, want the word it could not place", left)
	}
}

// TestWhatIsLeftOverIsHandedBack. Printing the settings for a reader who
// typed `orbit settings autopilot on` and walking away is how a run sat at a
// gate for ten minutes waiting for an autopilot nobody had turned on.
func TestWhatIsLeftOverIsHandedBack(t *testing.T) {
	args := map[string]string{}

	left := Fill(takes(aName("key", true)), args, strings.Fields("autopilot on and then some"))

	if args["key"] != "autopilot" {
		t.Errorf("the key arrived as %q", args["key"])
	}

	if !slices.Equal(left, []string{"on", "and", "then", "some"}) {
		t.Errorf("it was left with %v, want everything the verb had nowhere for", left)
	}
}

// TestAVerbThatTakesNothingHandsItAllBack, rather than swallowing a line
// nobody asked it to read.
func TestAVerbThatTakesNothingHandsItAllBack(t *testing.T) {
	args := map[string]string{}

	rest := strings.Fields("what is this")
	if left := Fill(takes(), args, rest); !slices.Equal(left, rest) {
		t.Errorf("a verb with no fields kept %v", left)
	}

	if len(args) != 0 {
		t.Errorf("a verb with no fields filled in %v", args)
	}
}

// TestALineThatRunsOutFillsWhatItReachedAndNoMore. A reader who typed half a
// command has a verb that will refuse for want of a field, and a field
// filled with the empty string would be a verb that ran on nothing.
func TestALineThatRunsOutFillsWhatItReachedAndNoMore(t *testing.T) {
	args := map[string]string{}

	half := takes(aName("id", true), aName("repo", true), sentence("text"))

	left := Fill(half, args, []string{"ACME-1"})

	if args["id"] != "ACME-1" {
		t.Errorf("the id arrived as %q", args["id"])
	}

	if _, filled := args["repo"]; filled {
		t.Errorf("a field the line never reached was filled with %q", args["repo"])
	}

	if len(left) != 0 {
		t.Errorf("%v had nowhere to go", left)
	}
}

// TestNoWordIsInventedAndNoneIsLost.
//
// The law this grammar has to keep, over lines nobody wrote down: every word
// handed in comes back somewhere — in a field, or in what was left over —
// and nothing appears that was not typed. A rule that dropped a word
// silently is a note with a sentence missing from the middle of it, and
// there is no example that would have caught it.
func TestNoWordIsInventedAndNoneIsLost(t *testing.T) {
	shapes := [][]Field{
		{aName("id", true), sentence("text")},
		{aName("id", true), aName("repo", false), sentence("text")},
		{aName("key", true), aName("value", true)},
		{aName("task", false)},
		{sentence("text")},
		{},
	}

	vocabulary := []string{"ACME-1", "on", "--", "it", "needs", "a", "test", "-x", ""}

	seed := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // a shuffled table, not a secret

	for range 500 {
		shape := shapes[seed.IntN(len(shapes))]

		rest := make([]string, seed.IntN(6))
		for i := range rest {
			rest[i] = vocabulary[seed.IntN(len(vocabulary))]
		}

		// A leading "--" is the shell's and is meant to disappear, so it is
		// not part of what has to come back.
		want := slices.Clone(rest)
		if len(want) > 0 && want[0] == "--" {
			want = want[1:]
		}

		args := map[string]string{}
		left := Fill(takes(shape...), args, slices.Clone(rest))

		filled := strings.Join(slices.Sorted(maps.Values(args)), " ")
		over := strings.Join(left, " ")

		got := slices.Sorted(slices.Values(strings.Fields(filled + " " + over)))
		typed := slices.Sorted(slices.Values(strings.Fields(strings.Join(want, " "))))

		if !slices.Equal(got, typed) {
			t.Fatalf("over %v with %d fields: the words came back as %v, want %v",
				rest, len(shape), got, typed)
		}
	}
}
