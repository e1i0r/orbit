package learn

// Where a kept rule ends up: the folder the work was in, unless somebody
// says otherwise.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/store"
)

// aCheckout is a directory with two folders in it, which is the whole of
// what a place is read against.
func aCheckout(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()

	for _, folder := range []string{"internal/db", "internal/ui"} {
		if err := os.MkdirAll(filepath.Join(repo, folder), 0o750); err != nil {
			t.Fatalf("make %s: %v", folder, err)
		}
	}

	return repo
}

// filed is the checkout's own facts, which is where a placed rule lands: a
// rule about a folder travels with the clone it is about.
func filed(t *testing.T, s *store.Store, repo string) []knowledge.Fact {
	t.Helper()

	got, err := knowledge.NewStore(s.Root()).LoadRepo(repo)
	if err != nil {
		t.Fatalf("read what the checkout knows: %v", err)
	}

	return got
}

// waitingFromARun is one sentence said at a task, with the checkout and the
// folder the work was in.
func waitingFromARun(t *testing.T, s *store.Store, repo, path, text string) Said {
	t.Helper()

	one := said(1, text)
	one.About, one.Repo, one.Path = "ACME-40", repo, path

	if err := Propose(s, one); err != nil {
		t.Fatalf("propose: %v", err)
	}

	return one
}

// TestTheFolderTheWorkWasInIsWhereTheRuleGoes.
//
// A rule said in the middle of one folder is about that folder often enough
// that retyping the path is the small tax that ends with nobody placing
// rules at all. Nothing here reads the words: where somebody was working is
// known, and what they meant is not.
func TestTheFolderTheWorkWasInIsWhereTheRuleGoes(t *testing.T) {
	s := root(t)
	repo := aCheckout(t)

	one := waitingFromARun(t, s, repo, "internal/db", "every query is a named constant")

	if err := Keep(s, one.At, one.Text, "", Place{}); err != nil {
		t.Fatalf("keep: %v", err)
	}

	got := filed(t, s, repo)
	if len(got) != 1 {
		t.Fatalf("keeping it wrote %d facts", len(got))
	}

	if got[0].Scope.Kind != knowledge.Dir || got[0].Scope.Path != "internal/db" {
		t.Errorf("it was filed as %+v", got[0].Scope)
	}
}

// TestSayingAPlaceBeatsTheOneTheWorkWasIn, because the folder is where
// somebody was standing and not a reading of what they meant. Agreeing with
// it has to be as cheap as typing over it, and typing over it has to win.
func TestSayingAPlaceBeatsTheOneTheWorkWasIn(t *testing.T) {
	s := root(t)
	repo := aCheckout(t)

	one := waitingFromARun(t, s, repo, "internal/db", "the window speaks in two languages")

	if err := Keep(s, one.At, one.Text, "", Place{Path: "internal/ui"}); err != nil {
		t.Fatalf("keep: %v", err)
	}

	got := filed(t, s, repo)
	if got[0].Scope.Path != "internal/ui" {
		t.Errorf("it was filed in %q", got[0].Scope.Path)
	}
}

// TestADotIsHowSomebodySaysTheWholeCheckout.
//
// Half of what gets said at a run is not about the folder it was said in —
// "never merge without the tests passing" is about the project. Without a
// way to say so out loud, the only way back out of a place that arrived
// filled in would be to leave the rule where it does not belong.
func TestADotIsHowSomebodySaysTheWholeCheckout(t *testing.T) {
	s := root(t)
	repo := aCheckout(t)

	one := waitingFromARun(t, s, repo, "internal/db", "never merge without the tests passing")

	if err := Keep(s, one.At, one.Text, "", Place{Path: "."}); err != nil {
		t.Fatalf("keep: %v", err)
	}

	got := filed(t, s, repo)
	if got[0].Scope.Kind != knowledge.Repo || got[0].Scope.Path != "" {
		t.Errorf("it was filed as %+v", got[0].Scope)
	}
}

// TestASentenceFromNoFolderIsAboutTheWholeCheckout, which is what a run that
// has changed nothing yet, or changed things all over, comes back with.
func TestASentenceFromNoFolderIsAboutTheWholeCheckout(t *testing.T) {
	s := root(t)
	repo := aCheckout(t)

	one := waitingFromARun(t, s, repo, "", "the commits are written in English")

	if err := Keep(s, one.At, one.Text, "", Place{}); err != nil {
		t.Fatalf("keep: %v", err)
	}

	got := filed(t, s, repo)
	if got[0].Scope.Kind != knowledge.Repo {
		t.Errorf("it was filed as %+v", got[0].Scope)
	}
}

// TestAgreeingWithAModelDoesNotMakeItYours.
//
// The screen that lists facts says where each one came from, and the whole
// of why a fact can be trusted is that it can be traced back to where it
// actually came from. A sentence an engine worked out mid-task stays the
// engine's finding after somebody says yes to it; saying yes is agreement,
// not authorship.
func TestAgreeingWithAModelDoesNotMakeItYours(t *testing.T) {
	s := root(t)
	repo := aCheckout(t)

	one := said(1, "the fuzz tests hang when the seed is fixed")
	one.By, one.About, one.Repo = AModel, "ACME-40", repo

	if err := Propose(s, one); err != nil {
		t.Fatalf("propose: %v", err)
	}

	if err := Keep(s, one.At, one.Text, "", Place{}); err != nil {
		t.Fatalf("keep: %v", err)
	}

	got := filed(t, s, repo)[0]
	if got.Source != knowledge.FromRecord {
		t.Errorf("a rule the model found is recorded as coming from %v", got.Source)
	}

	// And the run it came out of travels with it: a sentence nobody can
	// trace back to the task that produced it is one the model may as well
	// have made up.
	if got.Ref != "ACME-40" {
		t.Errorf("it says it came from %q", got.Ref)
	}
}

// TestWhatAPersonSaysIsStillTheirs, which is the other half of the same
// rule: everything that is not an engine working alone is somebody talking.
func TestWhatAPersonSaysIsStillTheirs(t *testing.T) {
	s := root(t)
	repo := aCheckout(t)

	one := waitingFromARun(t, s, repo, "", "never merge without the tests passing")

	if err := Keep(s, one.At, one.Text, "", Place{}); err != nil {
		t.Fatalf("keep: %v", err)
	}

	if got := filed(t, s, repo)[0]; got.Source != knowledge.Human {
		t.Errorf("a rule somebody said is recorded as coming from %v", got.Source)
	}
}

// TestTheTrayNeverPrintsAModelTheSameAsAPerson.
//
// A sentence an engine worked out on its own is read more carefully than one
// somebody said, and a reader who cannot tell them apart is a reader
// agreeing with both at the same speed.
func TestTheTrayNeverPrintsAModelTheSameAsAPerson(t *testing.T) {
	mine := Said{By: Operator, About: "ACME-40"}
	if got := mine.From(); got != "ACME-40" {
		t.Errorf("a sentence somebody said at a task reads %q", got)
	}

	its := Said{By: AModel, About: "ACME-40"}
	if got := its.From(); got == mine.From() {
		t.Errorf("a model's finding and a person's words both read %q", got)
	}

	if got := its.From(); !strings.Contains(got, "ACME-40") || !strings.Contains(got, AModel) {
		t.Errorf("a model's finding reads %q, which does not say both", got)
	}
}
