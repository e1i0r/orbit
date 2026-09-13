package learn

// A sentence waiting, and what becomes of it.

import (
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/store"
)

// root is a state root with a record under it.
func root(t *testing.T) *store.Store {
	t.Helper()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	t.Cleanup(func() { _ = s.Close() })

	return s
}

// said is one sentence, at a moment of its own.
func said(n int, text string) Said {
	return Said{At: time.Date(2026, 9, 11, 9, n, 0, 0, time.UTC), Text: text}
}

// facts is what Orbit knows.
func facts(t *testing.T, s *store.Store) []knowledge.Fact {
	t.Helper()

	got, err := knowledge.NewStore(s.Root()).Load("")
	if err != nil {
		t.Fatalf("read what Orbit knows: %v", err)
	}

	return got
}

// TestASentenceWaitsWhereTheModelCannotSeeIt.
//
// Every fact there is goes into every phase's prompt. A sentence nobody has
// agreed to yet kept as a fact would be a rule the model was told about
// because somebody typed it once.
func TestASentenceWaitsWhereTheModelCannotSeeIt(t *testing.T) {
	s := root(t)

	if err := Propose(s, said(1, "never push without the tests passing")); err != nil {
		t.Fatalf("propose: %v", err)
	}

	waiting, err := Waiting(s)
	if err != nil {
		t.Fatalf("waiting: %v", err)
	}

	if len(waiting) != 1 {
		t.Fatalf("%d sentences are waiting, want the one just said", len(waiting))
	}

	if got := facts(t, s); len(got) != 0 {
		t.Errorf("Orbit already knows %d things, and nobody has agreed to anything", len(got))
	}
}

// TestTheSameSentenceIsOnlyOfferedOnce.
//
// Whoever fills this reads a whole thread every time, so most of what it
// finds it has found before. Offering it again would mean a tray that grows
// every time it is opened.
func TestTheSameSentenceIsOnlyOfferedOnce(t *testing.T) {
	s := root(t)

	one := said(1, "always run make check first")

	for range 3 {
		if err := Propose(s, one); err != nil {
			t.Fatalf("propose: %v", err)
		}
	}

	waiting, err := Waiting(s)
	if err != nil {
		t.Fatalf("waiting: %v", err)
	}

	if len(waiting) != 1 {
		t.Errorf("one sentence said once is waiting %d times", len(waiting))
	}
}

// TestKeepingASentenceIsHowOrbitLearnsIt.
func TestKeepingASentenceIsHowOrbitLearnsIt(t *testing.T) {
	s := root(t)

	one := said(1, "never push without the tests passing")
	if err := Propose(s, one); err != nil {
		t.Fatalf("propose: %v", err)
	}

	if err := Keep(s, one.At, one.Text, "", ""); err != nil {
		t.Fatalf("keep: %v", err)
	}

	got := facts(t, s)
	if len(got) != 1 {
		t.Fatalf("Orbit knows %d things after keeping one", len(got))
	}

	if got[0].Phrase != one.Text {
		t.Errorf("Orbit learned %q, and you said %q", got[0].Phrase, one.Text)
	}

	if got[0].Source != knowledge.Human {
		t.Error("a sentence you kept is not recorded as yours")
	}

	waiting, err := Waiting(s)
	if err != nil {
		t.Fatalf("waiting: %v", err)
	}

	if len(waiting) != 0 {
		t.Errorf("%d sentences are still waiting after being kept", len(waiting))
	}
}

// TestCorrectingIsHowMostOfTheseAreAccepted.
//
// What you meant is what you typed the second time, and being asked is
// worth nothing if the only answers are yes and no.
func TestCorrectingIsHowMostOfTheseAreAccepted(t *testing.T) {
	s := root(t)

	one := said(1, "never push without tests")
	if err := Propose(s, one); err != nil {
		t.Fatalf("propose: %v", err)
	}

	better := "never open a pull request until make check is green"
	if err := Keep(s, one.At, better, "make check", ""); err != nil {
		t.Fatalf("keep: %v", err)
	}

	got := facts(t, s)
	if len(got) != 1 || got[0].Phrase != better {
		t.Fatalf("Orbit learned %v, want the sentence as it was corrected", got)
	}

	if got[0].Action() != knowledge.Stops {
		t.Error("a rule given a command that can answer yes or no does not refuse work")
	}
}

// TestARuleWithNoCommandAdvisesRatherThanRefuses, which is what most of them
// will be: a sentence is a sentence, and refusing needs something that can
// say no without an opinion in it.
func TestARuleWithNoCommandAdvisesRatherThanRefuses(t *testing.T) {
	s := root(t)

	one := said(1, "keep the sentences short")
	if err := Propose(s, one); err != nil {
		t.Fatalf("propose: %v", err)
	}

	if err := Keep(s, one.At, one.Text, "", ""); err != nil {
		t.Fatalf("keep: %v", err)
	}

	if got := facts(t, s); got[0].Action() != knowledge.Warns {
		t.Error("a rule with no command claims to refuse work")
	}
}

// TestDroppingLeavesTheSentenceWhereYouSaidIt.
func TestDroppingLeavesTheSentenceWhereYouSaidIt(t *testing.T) {
	s := root(t)

	one := said(1, "never mind, that was not a rule")
	if err := Propose(s, one); err != nil {
		t.Fatalf("propose: %v", err)
	}

	if err := Drop(s, one.At); err != nil {
		t.Fatalf("drop: %v", err)
	}

	waiting, err := Waiting(s)
	if err != nil {
		t.Fatalf("waiting: %v", err)
	}

	if len(waiting) != 0 {
		t.Errorf("%d sentences are waiting after being dropped", len(waiting))
	}

	if got := facts(t, s); len(got) != 0 {
		t.Errorf("Orbit learned %d things from a sentence you said was not a rule", len(got))
	}
}

// TestAnsweringTwiceIsRefused.
//
// A screen clicked twice, a command run twice, two windows open on the same
// tray. One question has one answer.
func TestAnsweringTwiceIsRefused(t *testing.T) {
	s := root(t)

	one := said(1, "always wrap errors")
	if err := Propose(s, one); err != nil {
		t.Fatalf("propose: %v", err)
	}

	if err := Keep(s, one.At, one.Text, "", ""); err != nil {
		t.Fatalf("keep: %v", err)
	}

	if err := Drop(s, one.At); err == nil {
		t.Error("a sentence already kept was dropped as well")
	}

	if got := facts(t, s); len(got) != 1 {
		t.Errorf("Orbit knows %d things after one sentence was answered twice", len(got))
	}
}

// TestASentenceThatSaysNothingIsRefused, because the tray's text is
// editable and an empty box is a way to write a fact with no sentence in it.
func TestASentenceThatSaysNothingIsRefused(t *testing.T) {
	s := root(t)

	one := said(1, "always wrap errors")
	if err := Propose(s, one); err != nil {
		t.Fatalf("propose: %v", err)
	}

	if err := Keep(s, one.At, "   ", "", ""); err == nil {
		t.Error("a rule with no sentence was written down")
	}

	waiting, err := Waiting(s)
	if err != nil {
		t.Fatalf("waiting: %v", err)
	}

	if len(waiting) != 1 {
		t.Error("a refused sentence left the tray")
	}
}
