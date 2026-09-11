package learn

// Which sentences read as somebody laying down a rule.

import (
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/store"
)

// TestARuleIsNoticedInEitherLanguage.
//
// The supervisor is talked to in whichever language somebody thinks in, and
// a reader whose rules are only noticed in English is a reader Orbit does
// not listen to.
func TestARuleIsNoticedInEitherLanguage(t *testing.T) {
	rules := []string{
		"never push a pull request without the tests passing",
		"always wrap errors with %w",
		"don't add dependencies without asking me",
		"do not touch the migrations",
		"make sure the coverage stays above 90",
		"from now on write the test in the same commit",
		"i want the PR description in English",
		"i don't want you pushing to main",
		"you must run make check before opening anything",
		"please always run the linter",
		"nunca pushees un PR sin validar tests",
		"siempre envolvé los errores",
		"no quiero que toques las migraciones",
		"quiero que el PR esté en inglés",
		"tenés que correr make check antes",
		"hay que dejar el coverage arriba de 90",
		"de ahora en más los tests van en el mismo commit",
		"por favor nunca toques la rama main",
	}

	for _, said := range rules {
		if !aboutTheFuture(said) {
			t.Errorf("not noticed as a rule: %q", said)
		}
	}
}

// TestOrdinaryTalkIsNotARule.
//
// This is the half that decides whether the tray is worth opening. Every
// false one is a row somebody has to dismiss, and a tray that fills up with
// conversation is a tray nobody reads.
func TestOrdinaryTalkIsNotARule(t *testing.T) {
	talk := []string{
		"what happened with ACME-3?",
		"should i always run the tests first?",
		"the build broke again",
		"we never found out why the fuzz test hangs",
		"ok, thanks",
		"retry the webhook on 5xx and open a PR",
		"qué pasó con la tarea de ayer",
		"se rompió el build otra vez",
		"dale, seguí",
		"",
		"   ",
	}

	for _, said := range talk {
		if aboutTheFuture(said) {
			t.Errorf("read as a rule and it is not: %q", said)
		}
	}
}

// TestAParagraphIsNotARule.
//
// A rule is the thing somebody would have typed into a config file if there
// had been one. Past that it is a briefing, and reading briefings as rules
// fills the tray with everything anybody ever said.
func TestAParagraphIsNotARule(t *testing.T) {
	long := "always keep in mind that this repository has a long history and " +
		"the reason the ledger module looks the way it does is that we tried " +
		"three other shapes first, none of which survived contact with the " +
		"reporting requirements, so before changing anything in there please " +
		"read the decisions and then come back and tell me what you think"

	if aboutTheFuture(long) {
		t.Error("a paragraph was read as a rule")
	}
}

// TestWhatTheSupervisorSaysIsNotWhatYouSaid.
//
// Its own answers are recorded as the same sort of turn as yours, and it
// answers in sentences that look exactly like rules — because it is telling
// an agent what to do. Read as yours, the tray would fill with Orbit
// quoting itself back at you.
func TestWhatTheSupervisorSaysIsNotWhatYouSaid(t *testing.T) {
	s := root(t)

	at := time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC)

	Heard(s, "supervisor", at, "always run make check before opening a PR")
	Heard(s, "orbit", at.Add(time.Minute), "never push without the tests passing")

	waiting, err := Waiting(s)
	if err != nil {
		t.Fatalf("waiting: %v", err)
	}

	if len(waiting) != 1 {
		t.Fatalf("%d sentences are waiting, want only the one you said", len(waiting))
	}

	if waiting[0].Text != "never push without the tests passing" {
		t.Errorf("the tray holds %q", waiting[0].Text)
	}
}

// TestEveryWayInIsYou.
//
// The cockpit, a command and a tool call are all somebody talking. A channel
// nobody has invented yet counts as a person too, which is the way round
// that fails safe: an extra row to dismiss, rather than a rule silently not
// noticed.
func TestEveryWayInIsYou(t *testing.T) {
	s := root(t)

	at := time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC)
	for i, channel := range []string{"orbit", "mcp", "telegram"} {
		Heard(s, channel, at.Add(time.Duration(i)*time.Minute), "never force-push")
	}

	waiting, err := Waiting(s)
	if err != nil {
		t.Fatalf("waiting: %v", err)
	}

	if len(waiting) != 3 {
		t.Errorf("%d of three ways in were heard", len(waiting))
	}
}

// TestSayingSomethingStillWorksWhenTheTrayDoesNot.
//
// What somebody said is the fact about that moment. Whether it was also a
// rule is a question asked about it, and a question that cannot be asked is
// not a reason to lose the sentence.
func TestSayingSomethingStillWorksWhenTheTrayDoesNot(t *testing.T) {
	var nothing *store.Store

	Heard(nothing, "orbit", time.Now(), "never push without the tests passing")
}

// TestARuleSaidToTheSupervisorReachesTheTray, end to end through the one
// door every line of the thread goes through.
func TestARuleSaidToTheSupervisorReachesTheTray(t *testing.T) {
	s := root(t)

	// Written the way internal/supervisor writes it, which is what the hook
	// sits on: the same kind for every turn, the channel saying who.
	Heard(s, "orbit", time.Now().UTC(), "never open a PR before make check is green")

	waiting, err := Waiting(s)
	if err != nil {
		t.Fatalf("waiting: %v", err)
	}

	if len(waiting) != 1 {
		t.Fatalf("%d sentences reached the tray", len(waiting))
	}

	if err := Keep(s, waiting[0].At, waiting[0].Text, ""); err != nil {
		t.Fatalf("keep: %v", err)
	}

	if got := facts(t, s); len(got) != 1 || got[0].Source != knowledge.Human {
		t.Errorf("Orbit learned %v", got)
	}
}
