package learn

// The words this package writes things down in, and the two doors nothing
// was holding.
//
// Small functions, and every one of them a translation between two
// vocabularies: where a sentence came from into what a fact records as its
// source, a state into what the record calls moving to it, a scope into the
// words the rest of Orbit says it in. A translation nobody checks is the one
// that quietly maps two things onto one.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/db"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/store"
)

// TestWhereASentenceCameFromBecomesWhatAFactRecords. What changes how
// carefully a rule is read is whether somebody said it or something deduced
// it, so the four sources are kept apart and everything else is a person.
func TestWhereASentenceCameFromBecomesWhatAFactRecords(t *testing.T) {
	cases := []struct {
		by   string
		want knowledge.Source
	}{
		{AModel, knowledge.FromRecord},
		{FromAPaper, knowledge.FromDocs},
		{FromTheHistory, knowledge.FromHistory},
		{FromTheGates, knowledge.FromGates},
		{Operator, knowledge.Human},
		{"telegram", knowledge.Human},
		{"", knowledge.Human},
	}

	for _, c := range cases {
		if got := foundBy(Said{By: c.by}); got != c.want {
			t.Errorf("a sentence from %q is recorded as %v, want %v", c.by, got, c.want)
		}
	}
}

// TestMovingBackToApplyingIsOneThingHoweverItGotAway. A rule that was paused
// and one that was switched off both come back the same way, and which it
// was is the row before this one.
func TestMovingBackToApplyingIsOneThingHoweverItGotAway(t *testing.T) {
	cases := []struct {
		state knowledge.State
		want  string
	}{
		{knowledge.Paused, db.RulePaused},
		{knowledge.Off, db.RuleOff},
		{knowledge.Active, db.RuleOn},
	}

	for _, c := range cases {
		if got := Stood(c.state); got != c.want {
			t.Errorf("moving to %v is written down as %q, want %q", c.state, got, c.want)
		}
	}
}

// TestAScopeIsWrittenDownInTheWordsTheRestOfOrbitSaysItIn. The history is
// read beside the rest of the window, and a scope spelled two ways is two
// things a reader has to learn are one.
func TestAScopeIsWrittenDownInTheWordsTheRestOfOrbitSaysItIn(t *testing.T) {
	cases := []struct {
		name  string
		scope knowledge.Scope
		want  string
	}{
		{"everywhere", knowledge.Scope{Kind: knowledge.General}, "everywhere"},
		{"one language", knowledge.Scope{Kind: knowledge.Language, Lang: "go"}, "go"},
		{"a checkout", knowledge.Scope{Kind: knowledge.Repo, Repo: "/w/orbit"}, "the whole checkout"},
		{
			"one symbol",
			knowledge.Scope{Kind: knowledge.Symbol, Path: "internal/db/append.go", Symbol: "Append"},
			"internal/db/append.go#Append",
		},
		{"a folder", knowledge.Scope{Kind: knowledge.Dir, Path: "internal/db"}, "internal/db"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := about(c.scope); got != c.want {
				t.Errorf("it is written down as %q, want %q", got, c.want)
			}
		})
	}
}

// TestWhereASentenceCameFromAndWhatItIsAboutAreOneField. A topic nobody can
// see is a topic nobody can tell has gone wrong, and the whole reason the
// model chooses from a fixed list is that two readings of the same person
// should add up to something visible.
func TestWhereASentenceCameFromAndWhatItIsAboutAreOneField(t *testing.T) {
	if got := with("operator", "testing"); !strings.Contains(got, "operator") ||
		!strings.Contains(got, "testing") {
		t.Errorf("it reads %q, want both where it came from and what it is about", got)
	}

	// A sentence somebody said outright carries no topic, and a separator
	// with nothing after it is a field that looks broken.
	if got := with("operator", ""); got != "operator" {
		t.Errorf("a sentence with no topic reads %q, want just where it came from", got)
	}
}

// TestARuleRefusingToBeForgottenSaysWhatItDid. "No" on its own leaves
// somebody wondering whether the rule is special or the verb is broken, and
// the sentence a reader sees is built from this in their own language.
func TestARuleRefusingToBeForgottenSaysWhatItDid(t *testing.T) {
	err := DidSomethingError{What: Paused}

	if !strings.Contains(err.Error(), Paused) {
		t.Errorf("the refusal is %q, want it to say what the rule did", err)
	}

	if !strings.Contains(err.Error(), "history") {
		t.Errorf("the refusal is %q, want it to say the rule has one", err)
	}
}

// TestAQuestionAlreadySettledIsNotAskedAgain. It is a door because the four
// ways in cannot reach the record, and what it stops is the same everywhere:
// offering a reading somebody has already answered.
func TestAQuestionAlreadySettledIsNotAskedAgain(t *testing.T) {
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}

	defer func() { _ = s.Close() }() //nolint:errcheck // the test is over

	mark := FromTheHistory + ":tests travel:internal/db"

	answered, err := Answered(s, mark)
	if err != nil {
		t.Fatalf("ask whether it was offered: %v", err)
	}

	if answered {
		t.Error("a reading nobody has been offered reads as settled")
	}

	said := Said{
		At:    time.Date(2026, 9, 17, 9, 0, 1, 0, time.UTC),
		Text:  "a change here comes with a test",
		By:    FromTheHistory,
		Habit: mark,
	}
	if err := Propose(s, said); err != nil {
		t.Fatalf("offer it: %v", err)
	}

	answered, err = Answered(s, mark)
	if err != nil {
		t.Fatalf("ask again: %v", err)
	}

	if !answered {
		t.Error("a reading already offered reads as never offered")
	}

	// A reading with no mark on it is one nothing was drawn from, and the
	// empty name is not a mark every one of them shares.
	if got, err := Answered(s, ""); err != nil || got {
		t.Errorf("the empty mark reads as settled: %v, %v", got, err)
	}
}
