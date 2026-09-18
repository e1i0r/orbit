package cli

// The ports a rule moves through, and the one reading that spends money.
//
// All of them read zero. Replace and Forget are the same ports the window is
// handed — a rule corrected from a screen and one corrected from a command
// line have to move the same way and leave the same row behind in the record
// — and the reading that asks a model is where a bill nobody expected comes
// from if it picks an engine nobody chose.

import (
	"errors"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/store"
)

// aFact is one rule, written into the state root the world reads from.
func aFact(t *testing.T, s *store.Store, phrase string) knowledge.Rule {
	t.Helper()

	f := knowledge.Rule{
		Scope:  knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human,
		Phrase: phrase,
	}

	if _, err := knowledge.NewStore(s.Root()).Save(f); err != nil {
		t.Fatalf("write down %q: %v", phrase, err)
	}

	kept, err := knowledge.NewStore(s.Root()).Load("")
	if err != nil {
		t.Fatalf("read back %q: %v", phrase, err)
	}

	if len(kept) != 1 {
		t.Fatalf("the state root holds %d rules, want the one just written", len(kept))
	}

	return kept[0]
}

// TestARuleCorrectedThroughThePortLeavesARowBehind. The record is what a
// rule has been through, and a correction with nothing written down is a
// sentence that changed and a history that says it never did.
func TestARuleCorrectedThroughThePortLeavesARowBehind(t *testing.T) {
	w, s, _ := worldOf(t)

	was := aFact(t, s, "the fuzz tests hang")

	now := was
	now.Phrase = "the fuzz tests hang on an empty corpus"

	if err := w.Replace(was, now, learn.Turn{}); err != nil {
		t.Fatalf("correct the rule: %v", err)
	}

	facts, err := w.Facts()
	if err != nil {
		t.Fatalf("read the rules back: %v", err)
	}

	if len(facts) != 1 || facts[0].Phrase != now.Phrase {
		t.Fatalf("the rules read back as %+v, want the corrected sentence", facts)
	}

	turns, err := learn.History(s, was.ID)
	if err != nil {
		t.Fatalf("read what happened to it: %v", err)
	}

	var reworded *learn.Turn

	for i, one := range turns {
		if one.What == learn.Reworded {
			reworded = &turns[i]
		}
	}

	if reworded == nil {
		t.Fatalf("a correction wrote no turn: %+v", turns)
	}

	if reworded.Was != was.Phrase {
		t.Errorf("the turn says it was %q, want the sentence it replaced", reworded.Was)
	}
}

// TestARuleThatDidNothingCanBeForgotten. The one thing here that loses
// something, and the row it leaves is the reason it is allowed at all:
// something vanishing with no trace is the only outcome worse than not being
// able to remove it.
func TestARuleThatDidNothingCanBeForgotten(t *testing.T) {
	w, s, _ := worldOf(t)

	f := aFact(t, s, "the fuzz tests hang")

	if err := w.Forget(f); err != nil {
		t.Fatalf("forget a rule that did nothing: %v", err)
	}

	facts, err := w.Facts()
	if err != nil {
		t.Fatalf("read the rules back: %v", err)
	}

	if len(facts) != 0 {
		t.Errorf("the rule is still there: %+v", facts)
	}

	turns, err := learn.History(s, f.ID)
	if err != nil {
		t.Fatalf("read what happened to it: %v", err)
	}

	if len(turns) == 0 || turns[len(turns)-1].What != learn.Forgotten {
		t.Errorf("forgetting it wrote %+v, want the row that says it existed", turns)
	}
}

// TestARuleWithAHistoryIsNotForgotten. A rule the record has anything else
// to say about cannot be taken off the disk, and the refusal names what it
// did rather than saying no.
func TestARuleWithAHistoryIsNotForgotten(t *testing.T) {
	w, s, _ := worldOf(t)

	f := aFact(t, s, "the fuzz tests hang")

	paused := learn.Turn{Rule: f.ID, What: learn.Paused, Was: "too noisy"}
	if err := learn.Happened(s, paused); err != nil {
		t.Fatalf("pause it: %v", err)
	}

	err := w.Forget(f)
	if err == nil {
		t.Fatal("a rule with a history was forgotten")
	}

	var did learn.DidSomethingError
	if !errors.As(err, &did) {
		t.Fatalf("it was refused with %q, want the refusal that names what it did", err)
	}

	if !strings.Contains(err.Error(), learn.Paused) {
		t.Errorf("the refusal is %q, want it to say the rule was paused", err)
	}

	facts, err := w.Facts()
	if err != nil {
		t.Fatalf("read the rules back: %v", err)
	}

	if len(facts) != 1 {
		t.Errorf("a refused forget removed the rule anyway: %+v", facts)
	}
}

// TestAReadingWithNoEngineNamedAndNoneSetAsksRatherThanPicking. This is the
// one reading that spends money with no task behind it, and an engine nobody
// chose is a bill nobody expected.
func TestAReadingWithNoEngineNamedAndNoneSetAsksRatherThanPicking(t *testing.T) {
	w, _, _ := worldOf(t)

	_, err := w.reading("")
	if err == nil {
		t.Fatal("a reading picked an engine nobody chose")
	}

	for _, want := range []string{"-engine", "settings set engine"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal is %q, want it to offer %q", err, want)
		}
	}
}

// TestAnEngineThisBuildHasNotGotIsNamedAndSoAreTheOnesItHas. A reader told
// only "no" has to guess whether they mistyped it or the build is older than
// they thought.
func TestAnEngineThisBuildHasNotGotIsNamedAndSoAreTheOnesItHas(t *testing.T) {
	w, _, _ := worldOf(t)

	_, err := w.reading("gpt")
	if err == nil {
		t.Fatal("an engine this build has not got was chosen")
	}

	if !strings.Contains(err.Error(), "gpt") {
		t.Errorf("the refusal is %q, want it to say back the name that was asked for", err)
	}

	for _, has := range engineNames(newEngines()) {
		if !strings.Contains(err.Error(), has) {
			t.Errorf("the refusal is %q, want %q among the ones it has", err, has)
		}
	}
}

// TestTheEngineSomebodySetIsTheOneThatReads. Named or set, and never
// whichever happens to be installed.
func TestTheEngineSomebodySetIsTheOneThatReads(t *testing.T) {
	w, s, _ := worldOf(t)

	if got := w.settingsEngine(); got != "" {
		t.Errorf("a machine with nothing set reads with %q, want nothing", got)
	}

	cfg, err := s.Settings()
	if err != nil {
		t.Fatalf("read the settings: %v", err)
	}

	cfg.Engine = "claude"
	if err := s.SaveSettings(cfg); err != nil {
		t.Fatalf("write the settings: %v", err)
	}

	if got := w.settingsEngine(); got != "claude" {
		t.Errorf("it reads with %q, want the engine that was set", got)
	}

	// And an engine that is set but not installed is refused by name: that
	// is a fact about this machine, and a reader told which one is missing
	// can do something about it.
	if _, err := w.reading(""); err != nil && !strings.Contains(err.Error(), "claude") {
		t.Errorf("the refusal is %q, want it to name the engine that is set", err)
	}
}
