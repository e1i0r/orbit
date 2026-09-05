package knowledge

// What a fact is, what it is allowed to claim, and which ones reach a file.

import (
	"slices"
	"testing"
)

func humanFact(s Scope, phrase string) Fact {
	return Fact{Scope: s, Source: Human, Phrase: phrase}
}

// TestAFactThatCannotCheckItselfOnlyWarns is the honesty of the whole thing.
//
// "Warns" is a sentence in the context and needs nothing else. "Stops" is the
// gate refusing the work, and refusing needs something that answers yes or no
// without opinion — a command, a pattern over the diff, a test. A fact that
// asks to stop and brings no check would be a rule that never fires while
// reading as though it did, which is worse than no rule.
func TestAFactThatCannotCheckItselfOnlyWarns(t *testing.T) {
	asked := Fact{
		Scope:  Scope{Kind: Repo, Repo: "/w/orbit"},
		Source: Human,
		Phrase: "no UPDATE in ledger",
		Stops:  true,
	}

	if asked.Action() != Warns {
		t.Error("a fact asked to stop, with no check, claims to stop")
	}

	withCheck := asked
	withCheck.Check = "! git diff | grep -q 'UPDATE ledger'"

	if withCheck.Action() != Stops {
		t.Error("a fact asked to stop, with a check, only warns")
	}
}

// TestAFactWithoutASourceOrAPhraseIsRefused. From FRA-47: every fact has a
// source and a scope, and without a file behind it, it does not get in.
func TestAFactWithoutASourceOrAPhraseIsRefused(t *testing.T) {
	for what, f := range map[string]Fact{
		"no phrase": {Scope: Scope{Kind: General}, Source: Human},
		"no source": {Scope: Scope{Kind: General}, Phrase: "the PRs are in English"},
		"a language scope naming no language": {
			Scope: Scope{Kind: Language}, Source: Human, Phrase: "never discard an error",
		},
		"a file scope naming no repository": {
			Scope: Scope{Kind: File, Path: "a.go"}, Source: Human, Phrase: "careful",
		},
	} {
		if err := f.Validate(); err == nil {
			t.Errorf("a fact with %s was accepted", what)
		}
	}
}

// TestTheFactsOfAFileArriveWidestFirst. The agent reads them in order, so
// the last word belongs to whatever was written about the file itself.
func TestTheFactsOfAFileArriveWidestFirst(t *testing.T) {
	all := []Fact{
		humanFact(Scope{Kind: Dir, Repo: "/w/orbit", Path: "internal"}, "of internal"),
		humanFact(Scope{Kind: General}, "of everything"),
		humanFact(Scope{Kind: File, Repo: "/w/orbit", Path: "internal/ui/bar.go"}, "of the file"),
		humanFact(Scope{Kind: Language, Lang: "go"}, "of Go"),
		humanFact(Scope{Kind: Repo, Repo: "/w/orbit"}, "of the repository"),
		humanFact(Scope{Kind: Dir, Repo: "/w/orbit", Path: "internal/ui"}, "of internal/ui"),
		humanFact(Scope{Kind: Repo, Repo: "/w/payments"}, "of somebody else"),
		humanFact(Scope{Kind: Language, Lang: "ts"}, "of TypeScript"),
	}

	got := For(Target{Repo: "/w/orbit", Path: "internal/ui/bar.go"}, all)

	said := make([]string, 0, len(got))
	for _, f := range got {
		said = append(said, f.Phrase)
	}

	want := []string{"of everything", "of Go", "of the repository", "of internal", "of internal/ui", "of the file"}
	if !slices.Equal(said, want) {
		t.Errorf("the file was told %v,\nwant %v", said, want)
	}
}

// TestAFactThatIsOffIsNotTold. Turning one off is how somebody disagrees
// with it without losing the record that it was ever there.
func TestAFactThatIsOffIsNotTold(t *testing.T) {
	off := humanFact(Scope{Kind: General}, "of everything")
	off.Off = true

	if got := For(Target{Repo: "/w/orbit", Path: "a.go"}, []Fact{off}); len(got) != 0 {
		t.Errorf("a fact that was turned off was still told: %v", got)
	}
}

// TestEveryKeepsWhatWasTurnedOff, because the screen that lists facts is the
// one where a fact is turned back on: dropping them there would leave a file
// on disk that nothing in the window admits exists.
func TestEveryKeepsWhatWasTurnedOff(t *testing.T) {
	all := []Fact{
		{Scope: Scope{Kind: Repo, Repo: "/w/api"}, Phrase: "narrow", Source: Human},
		{Scope: Scope{Kind: General}, Phrase: "off and wide", Source: Human, Off: true},
	}

	got := Every(all)
	if len(got) != 2 {
		t.Fatalf("Every kept %d of 2 facts", len(got))
	}

	// Widest first, the same order everything else reads them in.
	if got[0].Phrase != "off and wide" {
		t.Errorf("Every put %q first", got[0].Phrase)
	}

	if kept := InScope(all); len(kept) != 1 {
		t.Errorf("InScope told a phase %d facts, want only the one that is on", len(kept))
	}
}
