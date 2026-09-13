package knowledge

// Where a rule stands, and what that decides.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOnlyAnActiveRuleReachesAPhase.
//
// A rule used to be said or not said, and that one switch is what forced
// switching a rule off when what was needed was something else: "skip this
// one while we get the coverage up" is not disagreeing with it. Three states
// only earn their place if the two that are not active both stay out of the
// prompt.
func TestOnlyAnActiveRuleReachesAPhase(t *testing.T) {
	all := []Fact{
		{Scope: Scope{Kind: General}, Source: Human, Phrase: "applying"},
		{
			Scope: Scope{Kind: General}, Source: Human, Phrase: "paused", State: Paused,
			Why: "while we get the coverage up",
		},
		{Scope: Scope{Kind: General}, Source: Human, Phrase: "switched off", State: Off},
	}

	told := InScope(all)
	if len(told) != 1 || told[0].Phrase != "applying" {
		t.Errorf("a phase is told %d rules: %+v", len(told), told)
	}

	// And the screen that lists them shows all three, because a rule
	// nothing admits exists is one nobody can turn back on.
	if seen := Every(all); len(seen) != 3 {
		t.Errorf("the listing shows %d of them", len(seen))
	}
}

// TestAPausedRuleRemembersWhy.
//
// A pause with no reason is a switch, and a switch is the thing this was
// added to stop being the only answer. The reason is what somebody reads
// when they come back, and the only thing that will tell them whether it
// made sense.
func TestAPausedRuleRemembersWhy(t *testing.T) {
	s, repo := aRepo(t)

	if _, err := s.Save(Fact{
		Scope: Scope{Kind: Repo, Repo: repo}, Source: Human,
		Phrase: "coverage stays above 90%", State: Paused,
		Why: "the repo has never been past 80, we are fixing that first",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	got := only(t, s, repo)
	if got.State != Paused {
		t.Errorf("it came back at %v", got.State)
	}

	if !strings.Contains(got.Why, "never been past 80") {
		t.Errorf("it came back with the reason %q", got.Why)
	}
}

// TestARuleThatAppliesSaysNothingAboutItself.
//
// Applying is the ordinary case and the zero value. A header that spelled it
// out on every file would be a line somebody learns to stop reading, and a
// file written by hand with two lines in it has to work.
func TestARuleThatAppliesSaysNothingAboutItself(t *testing.T) {
	s, repo := aRepo(t)

	if _, err := s.Save(Fact{
		Scope: Scope{Kind: Repo, Repo: repo}, Source: Human, Phrase: "amounts are cents",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(repo, ".orbit", "knowledge", "amounts-are-cents.md"))
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(body), keyState) {
		t.Errorf("a rule that applies says so in its header:\n%s", body)
	}

	if got := only(t, s, repo); !got.Tells() {
		t.Errorf("a rule that says nothing about itself stands at %v", got.State)
	}
}

// TestAFileWrittenBeforeThereWereStatesIsStillOff.
//
// `off: true` is what every file written until now says, and a rule somebody
// disagreed with quietly coming back on would be the worst possible way to
// find out this changed.
func TestAFileWrittenBeforeThereWereStatesIsStillOff(t *testing.T) {
	s, repo := aRepo(t)

	at := filepath.Join(repo, ".orbit", "knowledge", "old.md")
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(at,
		[]byte("---\nscope: repo\nsource: human\noff: true\n---\n\nnever push on a Friday\n"),
		0o600); err != nil {
		t.Fatal(err)
	}

	if got := only(t, s, repo); got.State != Off {
		t.Errorf("a rule written down as off came back at %v", got.State)
	}
}

// TestARuleWaitingForAnAnswerSaysSoAndStillApplies.
//
// Skipping a rule at a run leaves it applying and asks for a decision;
// pausing it stops it applying and asks for the same decision. Folded into
// one field, one of those two would have to lie about whether the rule is
// still in the prompt.
func TestARuleWaitingForAnAnswerSaysSoAndStillApplies(t *testing.T) {
	s, repo := aRepo(t)

	if _, err := s.Save(Fact{
		Scope: Scope{Kind: Repo, Repo: repo}, Source: Human,
		Phrase: "coverage stays above 90%", Review: true,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	got := only(t, s, repo)
	if !got.Review {
		t.Error("a rule waiting for an answer does not say so")
	}

	if !got.Tells() {
		t.Error("a rule that was skipped stopped applying")
	}
}
