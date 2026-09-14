package verb

// Deciding about a rule with everything it has put you through in front of
// you.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/words"
)

// wearing is the turns a rule that has been in somebody's way looks like.
func wearing(what ...string) []learn.Turn {
	at := time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC)

	turns := make([]learn.Turn, 0, len(what))
	for i, one := range what {
		turns = append(turns, learn.Turn{
			Rule: "aaaa1111", At: at.Add(time.Duration(i) * time.Hour),
			What: one, Phase: "test",
		})
	}

	return turns
}

// TestTheStoryIsAnArgumentAndNotACount.
//
// A rule that works perfectly never stops anything, so "it stopped the work
// zero times" means two opposite things and no number tells them apart. What
// a reader is doing here is deciding, and "it stopped the work four times in
// test and you got past it every time" is an argument.
func TestTheStoryIsAnArgumentAndNotACount(t *testing.T) {
	p := words.For("")
	f := aRule("aaaa1111", "coverage stays above 90%")

	told := strings.Join(Story(p, f, wearing(
		learn.Written, learn.Failed, learn.Skipped, learn.Failed, learn.Skipped)), "\n")

	if !strings.Contains(told, "test") || !strings.Contains(told, "every time") {
		t.Errorf("a rule somebody always gets past reads:\n%s", told)
	}

	// And the other way round: refused and fixed is a rule doing its job.
	fixed := strings.Join(Story(p, f, wearing(learn.Written, learn.Failed, learn.Failed)), "\n")
	if !strings.Contains(fixed, "fixed") {
		t.Errorf("a rule whose refusals were all fixed reads:\n%s", fixed)
	}
}

// TestARuleThatHasNotStoppedAnythingSaysSo, because silence is the good case
// and an empty review would read as one nobody had looked at.
func TestARuleThatHasNotStoppedAnythingSaysSo(t *testing.T) {
	p := words.For("")
	f := aRule("aaaa1111", "coverage stays above 90%")

	told := strings.Join(Story(p, f, wearing(learn.Written)), "\n")
	if !strings.Contains(told, "working") {
		t.Errorf("a rule nothing has happened to reads:\n%s", told)
	}
}

// TestNarrowingIsTheAnswerThatWasAlmostAlwaysWanted.
//
// A rule that annoys you in one folder and earns its keep in another is not a
// rule to switch off — it is a rule about the second folder that somebody
// wrote too wide. Until it could be moved, the only answer was to lose it.
func TestNarrowingIsTheAnswerThatWasAlmostAlwaysWanted(t *testing.T) {
	w := trayOf(t)
	repo := aCheckoutWith(t, "internal/db")

	was := aRule("aaaa1111", "coverage stays above 90%")
	was.Scope = knowledge.Scope{Kind: knowledge.Repo, Repo: repo}
	was.Review, was.State, was.Why = true, knowledge.Paused, "it is too wide"
	w.facts = []knowledge.Fact{was}

	if _, err := asked(t, w, "rules correct", map[string]string{
		"rule": "aaaa1111", "in": "internal/db",
	}); err != nil {
		t.Fatalf("correct: %v", err)
	}

	now := w.facts[0]
	if now.Scope.Kind != knowledge.Dir || now.Scope.Path != "internal/db" {
		t.Errorf("it now applies to %+v", now.Scope)
	}

	// Deciding about it is what stops it waiting, and it applies again:
	// narrowing a rule is keeping it, not shelving it.
	if now.Review || !now.Tells() || now.Why != "" {
		t.Errorf("after correcting it: review=%v, state=%v, why=%q", now.Review, now.State, now.Why)
	}
}

// TestWhatIsNotNamedIsLeftAlone, so that saying a rule better does not
// quietly drop the command that makes it stop the work.
func TestWhatIsNotNamedIsLeftAlone(t *testing.T) {
	w := trayOf(t)

	was := aRule("aaaa1111", "coverage stays above 90%")
	was.Check, was.Stops = "make coverage", true
	w.facts = []knowledge.Fact{was}

	if _, err := asked(t, w, "rules correct", map[string]string{
		"rule": "aaaa1111", "text": "coverage stays above 90% in this project",
	}); err != nil {
		t.Fatalf("correct: %v", err)
	}

	now := w.facts[0]
	if now.Check != "make coverage" || now.Action() != knowledge.Stops {
		t.Errorf("saying it better left it with check %q", now.Check)
	}

	// And naming an empty check does take it away, which is the other thing
	// somebody means when they type one.
	if _, err := asked(t, w, "rules correct", map[string]string{
		"rule": "aaaa1111", "check": "",
	}); err != nil {
		t.Fatalf("correct: %v", err)
	}

	if now := w.facts[0]; now.Check != "" || now.Action() == knowledge.Stops {
		t.Errorf("an emptied check left it stopping the work over %q", now.Check)
	}
}

// TestDecidingAgainstARuleKeepsIt. Disagreeing with a rule and losing the
// record that it existed are different things, and only the first happens.
func TestDecidingAgainstARuleKeepsIt(t *testing.T) {
	w := trayOf(t)

	was := aRule("aaaa1111", "coverage stays above 90%")
	was.Review = true
	w.facts = []knowledge.Fact{was}

	out, err := asked(t, w, "rules off", map[string]string{"rule": "aaaa1111"})
	if err != nil {
		t.Fatalf("off: %v", err)
	}

	if !strings.Contains(out.Said, "stays") {
		t.Errorf("switching it off answered %q", out.Said)
	}

	now := w.facts[0]
	if now.State != knowledge.Off || now.Tells() || now.Review {
		t.Errorf("it stands at %v, telling=%v, review=%v", now.State, now.Tells(), now.Review)
	}
}

// TestARuleAboutNoCheckoutHasNowhereToNarrowTo, and is refused in words
// rather than filed against a repository picked for somebody.
func TestARuleAboutNoCheckoutHasNowhereToNarrowTo(t *testing.T) {
	w := trayOf(t)
	w.facts = []knowledge.Fact{{
		ID: "aaaa1111", Source: knowledge.Human, Phrase: "PRs are written in English",
		Scope: knowledge.Scope{Kind: knowledge.General},
	}}

	_, err := asked(t, w, "rules correct", map[string]string{
		"rule": "aaaa1111", "in": "internal/db",
	})
	if err == nil {
		t.Fatal("a rule about everywhere was narrowed to a folder of nothing")
	}

	if !strings.Contains(err.Error(), "internal/db") {
		t.Errorf("the refusal reads %q", err)
	}
}

// aCheckoutWith is a directory with that folder in it, which is the whole of
// what a place is read against.
func aCheckoutWith(t *testing.T, folder string) string {
	t.Helper()

	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, folder), 0o750); err != nil {
		t.Fatalf("make %s: %v", folder, err)
	}

	return repo
}
