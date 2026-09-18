package cli

// The ports the window reaches a rule through.
//
// Six of them, and each read one line: the closure was built and never
// called. They are what the knowledge screen does — keep a sentence, drop
// one, read what a checkout already refuses work over, correct a rule,
// forget one, and tell its story — and a port that quietly wrote the wrong
// thing would show up as a screen that agrees with itself and not with the
// disk.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// aStore is a state root of the test's own, opened once.
func aStore(t *testing.T) *store.Store {
	t.Helper()

	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)

	s, err := store.New(home)
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}

	t.Cleanup(func() { _ = s.Close() }) //nolint:errcheck // the test is over

	return s
}

// saidAt is a sentence in the tray, waiting to be answered.
func saidAt(t *testing.T, s *store.Store, at time.Time, text string) {
	t.Helper()

	if err := learn.Propose(s, learn.Said{At: at, Text: text, By: learn.Operator}); err != nil {
		t.Fatalf("put %q in the tray: %v", text, err)
	}
}

// TestKeepingASentenceWritesItWhereTheScreenWasLeft. The supervisor's thread
// is about the board rather than about one task, so the sentence itself
// knows no checkout: the one the window was opened over is what it goes on.
func TestKeepingASentenceWritesItWhereTheScreenWasLeft(t *testing.T) {
	s := aStore(t)

	// A real directory, because a rule is filed at a place and a place that
	// is not in the checkout is refused — which is the point: a rule about
	// a folder nobody has is a rule that never applies.
	checkout := t.TempDir()
	if err := os.MkdirAll(filepath.Join(checkout, "internal", "db"), 0o750); err != nil {
		t.Fatalf("make the folder the rule is about: %v", err)
	}

	at := time.Date(2026, 9, 17, 9, 0, 1, 0, time.UTC)
	saidAt(t, s, at, "run the tests before you push")

	keep := keepRulePort(s, checkout)

	if err := keep(at, "run the tests before you push", "make check", "internal/db"); err != nil {
		t.Fatalf("keep it: %v", err)
	}

	facts, err := knowledge.NewStore(s.Root()).Load(checkout)
	if err != nil {
		t.Fatalf("read the rules back: %v", err)
	}

	if len(facts) != 1 {
		t.Fatalf("keeping one sentence wrote %d rules", len(facts))
	}

	one := facts[0]
	if one.Phrase != "run the tests before you push" {
		t.Errorf("the rule says %q", one.Phrase)
	}

	if one.Scope.Repo != checkout || one.Scope.Path != "internal/db" {
		t.Errorf("it was filed at %+v, want the checkout the screen was left with", one.Scope)
	}

	if one.Check != "make check" {
		t.Errorf("the gate is %q, want the command the sentence arrived with", one.Check)
	}

	// And it has left the tray: a sentence answered is not a question any
	// more.
	waiting, err := learn.Waiting(s)
	if err != nil {
		t.Fatalf("read the tray: %v", err)
	}

	if len(waiting) != 0 {
		t.Errorf("the sentence is still waiting: %+v", waiting)
	}
}

// TestDroppingASentenceLeavesItWhereItWasSaid. Saying it was not a rule is
// an answer, not a deletion: the thread is append-only and so is what
// somebody decided about it.
func TestDroppingASentenceLeavesItWhereItWasSaid(t *testing.T) {
	s := aStore(t)

	at := time.Date(2026, 9, 17, 9, 0, 1, 0, time.UTC)
	saidAt(t, s, at, "always use tabs")

	if err := dropRulePort(s)(at); err != nil {
		t.Fatalf("drop it: %v", err)
	}

	waiting, err := learn.Waiting(s)
	if err != nil {
		t.Fatalf("read the tray: %v", err)
	}

	if len(waiting) != 0 {
		t.Errorf("a dropped sentence is still waiting: %+v", waiting)
	}

	facts, err := knowledge.NewStore(s.Root()).Load("")
	if err != nil {
		t.Fatalf("read the rules: %v", err)
	}

	if len(facts) != 0 {
		t.Errorf("dropping a sentence wrote it down anyway: %+v", facts)
	}
}

// TestWhatACheckoutRefusesWorkOverComesBackAsACount. It reaches the disk and
// the record, which the window may not, and what the screen does next is
// read the tray again — so a number is the whole of the answer.
func TestWhatACheckoutRefusesWorkOverComesBackAsACount(t *testing.T) {
	s := aStore(t)

	root := t.TempDir()

	workflow := filepath.Join(root, ".github", "workflows", "ci.yml")
	if err := os.MkdirAll(filepath.Dir(workflow), 0o750); err != nil {
		t.Fatalf("make the workflow directory: %v", err)
	}

	body := "name: ci\non: [pull_request]\njobs:\n  check:\n    steps:\n      - run: make check\n"
	if err := os.WriteFile(workflow, []byte(body), 0o600); err != nil {
		t.Fatalf("write the workflow: %v", err)
	}

	gates := gatesPort(s)

	got, err := gates(root)
	if err != nil {
		t.Fatalf("read what the checkout refuses work over: %v", err)
	}

	if got != 1 {
		t.Fatalf("it found %d gates, want the one in the workflow", got)
	}

	// Asked again it finds nothing new: a sentence already put to somebody
	// is not put to them twice.
	again, err := gates(root)
	if err != nil {
		t.Fatalf("read it again: %v", err)
	}

	if again != 0 {
		t.Errorf("it offered %d the second time, want none", again)
	}
}

// TestTheKnowledgeScreenCorrectsARuleAboutNothingElse. That screen is opened
// over the board and not over a task, so a correction taken there is about
// the rule — the command line's own pause can say which run it was in the
// way at, because the reader typed it.
func TestTheKnowledgeScreenCorrectsARuleAboutNothingElse(t *testing.T) {
	s := aStore(t)

	was := kept(t, s, "the fuzz tests hang")

	now := was
	now.Phrase = "the fuzz tests hang on an empty corpus"

	if err := windowReplacePort(s)(was, now); err != nil {
		t.Fatalf("correct it: %v", err)
	}

	facts, err := knowledge.NewStore(s.Root()).Load("")
	if err != nil {
		t.Fatalf("read the rules back: %v", err)
	}

	if len(facts) != 1 || facts[0].Phrase != now.Phrase {
		t.Fatalf("the rules read back as %+v", facts)
	}

	turns, err := learn.History(s, was.ID)
	if err != nil {
		t.Fatalf("read what happened to it: %v", err)
	}

	var by string

	for _, one := range turns {
		if one.What == learn.Reworded {
			by = one.By
		}
	}

	if by != learn.Operator {
		t.Errorf("the correction was recorded as %q, want the person who took it", by)
	}
}

// TestForgettingARuleWithAHistorySaysWhatToDoInstead. internal/learn decides
// whether a rule can go and carries what it did in a typed error; the words
// are this layer's, because this is where the catalogue is.
func TestForgettingARuleWithAHistorySaysWhatToDoInstead(t *testing.T) {
	s := aStore(t)
	p := words.For("en")

	f := kept(t, s, "the fuzz tests hang")

	paused := learn.Turn{Rule: f.ID, What: learn.Paused, Was: "too noisy"}
	if err := learn.Happened(s, paused); err != nil {
		t.Fatalf("pause it: %v", err)
	}

	err := forgetRulePort(s, p)(f)
	if err == nil {
		t.Fatal("a rule with a history was forgotten")
	}

	for _, want := range []string{learn.Paused, "Switch it off"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal is %q, want %q in it", err, want)
		}
	}

	// And one that did nothing goes, which is the half this refusal exists
	// to leave alone.
	quiet := kept(t, s, "never log a card number")
	if err := forgetRulePort(s, p)(quiet); err != nil {
		t.Errorf("a rule that did nothing was not forgotten: %v", err)
	}
}

// TestARulesStoryIsToldInTheSameSentencesTheCommandLineTellsIt. Two surfaces
// telling it two ways would be two accounts of the evidence a person is
// about to decide on.
func TestARulesStoryIsToldInTheSameSentencesTheCommandLineTellsIt(t *testing.T) {
	s := aStore(t)
	p := words.For("en")

	f := kept(t, s, "the fuzz tests hang")

	told := ruleStoryPort(s, p)(f)
	if len(told) == 0 {
		t.Fatal("a rule that was kept has no story at all")
	}

	if !strings.Contains(strings.Join(told, " "), "kept it") {
		t.Errorf("the story is %v, want it to say when it was kept", told)
	}
}

// kept writes one rule down the way the tray does, and hands back the rule
// as the disk now holds it.
func kept(t *testing.T, s *store.Store, phrase string) knowledge.Rule {
	t.Helper()

	at := time.Now().UTC()
	saidAt(t, s, at, phrase)

	if err := keepRulePort(s, "")(at, phrase, "", ""); err != nil {
		t.Fatalf("keep %q: %v", phrase, err)
	}

	facts, err := knowledge.NewStore(s.Root()).Load("")
	if err != nil {
		t.Fatalf("read the rules: %v", err)
	}

	for _, one := range facts {
		if one.Phrase == phrase {
			return one
		}
	}

	t.Fatalf("%q was not written down: %+v", phrase, facts)

	return knowledge.Rule{}
}
