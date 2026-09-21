package verb

// Putting a setting back to what Orbit ships.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// TestEverySettingCanBePutBack is the other half of the promise the table
// makes. A setting somebody can choose and cannot unchoose is one they have
// to edit the file to be rid of, which is what `orbit settings` exists so
// nobody has to do.
func TestEverySettingCanBePutBack(t *testing.T) {
	// Something chosen for every key, so that clearing has something to
	// undo rather than passing on a struct that was already empty.
	chosen := store.Settings{
		Language: "es", Autopilot: true, UnreadCap: 40, Engine: "codex",
		Model: "sonnet", Flow: "careful", Theme: "tokyo-night", CheckRecord: true,
		ChatID:     "8477112",
		Notify:     true,
		BudgetTask: 9, BudgetWorkspace: 99, QuotaFloor: 15,
		Decisions: "on", DecisionFloor: 90,
	}

	for _, key := range settingKeys() {
		cfg := chosen

		was, now, err := blank(words.For("en"), &cfg, key)
		if err != nil {
			t.Fatalf("clear %s: %v", key, err)
		}

		if was == now {
			t.Errorf("clearing %s changed nothing: it was %q and it is %q", key, was, now)
		}

		// And what it is now is what a machine that never chose anything
		// reads. Nothing here invents a default of its own.
		if want := Fresh(key); now != want {
			t.Errorf("%s was put back to %q, and Orbit ships %q", key, now, want)
		}
	}
}

// TestClearingSomethingAlreadyDefaultIsNotAFailure. It is the nothing it
// looks like, and saying so is the answer — a reader who typed this because
// they were not sure what the setting held is owed that much.
func TestClearingSomethingAlreadyDefaultIsNotAFailure(t *testing.T) {
	cfg := store.Shipped()

	was, now, err := blank(words.For("en"), &cfg, "unread-cap")
	if err != nil {
		t.Fatalf("clear unread-cap: %v", err)
	}

	if was != now {
		t.Errorf("clearing a setting that was already there moved it: %q → %q", was, now)
	}
}

// TestFreshIsWhatAMachineWithNoSettingsReads holds the table and the store
// together. internal/store decides what a file that is not there answers
// with, and this table decides what each key comes as; a reader who has
// never opened settings and a reader who cleared one have to land in the
// same place.
func TestFreshIsWhatAMachineWithNoSettingsReads(t *testing.T) {
	shipped := store.Shipped()

	for _, s := range settingTable() {
		if got, want := Fresh(s.Name), s.Value(shipped); got != want {
			t.Errorf("Fresh(%q) is %q and the table reads %q", s.Name, got, want)
		}
	}

	if Fresh("colour") != "" {
		t.Errorf("a name the table does not have answered %q", Fresh("colour"))
	}
}

// TestARefusedKeyListsTheKeysThereAreWhenClearing, the same as setting one:
// silently doing nothing to a setting somebody believes they cleared is the
// worst of the three outcomes.
func TestARefusedKeyListsTheKeysThereAreWhenClearing(t *testing.T) {
	var cfg store.Settings

	_, _, err := blank(words.For("en"), &cfg, "colour")
	if err == nil {
		t.Fatal("clearing a setting that does not exist was allowed")
	}

	if !strings.Contains(err.Error(), "unread-cap") {
		t.Errorf("the refusal does not list the keys there are: %v", err)
	}
}

// TestEveryRuleIsWhole. The four closures are one decision about one setting
// — what it means, what it accepts, what it reads as and what it comes as —
// and a rule missing one of them is a rule that panics the first time
// somebody reaches for the half nobody wrote. It did, on the three budgets.
func TestEveryRuleIsWhole(t *testing.T) {
	for _, s := range settingTable() {
		switch {
		case s.About == nil:
			t.Errorf("%q says nothing about itself", s.Name)
		case s.Set == nil:
			t.Errorf("%q cannot be set", s.Name)
		case s.Value == nil:
			t.Errorf("%q cannot be read", s.Name)
		case s.Clear == nil:
			t.Errorf("%q cannot be put back", s.Name)
		}
	}
}

// TestClearingSaysWhatTheSettingIsNowAndWhetherItMoved.
//
// "Nothing to do" is an answer. A reader who typed this because they were
// not sure what the setting held is owed it — told only "cleared", they are
// no wiser than before, and a second reader looking over their shoulder
// cannot tell whether anything happened.
func TestClearingSaysWhatTheSettingIsNowAndWhetherItMoved(t *testing.T) {
	w := worldOf(t)

	// Something chosen, so clearing it has something to undo.
	mustAsk(t, w, "settings set", In{Args: map[string]string{"key": "language", "value": "es"}})

	out := mustAsk(t, w, "settings clear", In{Args: map[string]string{"key": "language"}})

	if !strings.Contains(out.Said, "language") {
		t.Errorf("it said %q, want the setting named", out.Said)
	}

	if strings.Contains(out.Said, "already") {
		t.Errorf("it said %q about a setting that had been chosen", out.Said)
	}

	// Cleared twice, the second says nothing moved rather than repeating
	// the first sentence.
	again := mustAsk(t, w, "settings clear", In{Args: map[string]string{"key": "language"}})
	if !strings.Contains(again.Said, "already") {
		t.Errorf("clearing a setting that was already back said %q", again.Said)
	}
}

// TestASettingThatComesAsNothingSaysSoInAWord. "engine is back to —" is a
// table cell in the middle of a sentence; a dash is what a listing draws and
// a word is what a sentence needs.
func TestASettingThatComesAsNothingSaysSoInAWord(t *testing.T) {
	w := worldOf(t)

	mustAsk(t, w, "settings set", In{Args: map[string]string{"key": "engine", "value": "codex"}})

	out := mustAsk(t, w, "settings clear", In{Args: map[string]string{"key": "engine"}})

	if strings.Contains(out.Said, "—") {
		t.Errorf("it said %q, want a word rather than the listing's dash", out.Said)
	}

	if !strings.Contains(out.Said, "nothing") {
		t.Errorf("it said %q, want it to say the setting comes as nothing", out.Said)
	}

	// And the listing draws the dash, which is the other half of the same
	// decision: an empty column reads as a table that failed to render.
	if got := unset(""); got != "—" {
		t.Errorf("the listing draws an empty setting as %q, want a dash", got)
	}

	if got := unset("codex"); got != "codex" {
		t.Errorf("the listing draws a chosen setting as %q", got)
	}
}

// TestClearingASettingNobodyDeclaredIsRefused, and the file is not rewritten
// on the way.
func TestClearingASettingNobodyDeclaredIsRefused(t *testing.T) {
	w := worldOf(t)

	if _, err := Run(ctxOf(), w, "settings clear", In{Args: map[string]string{"key": "gravity"}}); err == nil {
		t.Error("a setting nobody declared was cleared")
	}
}
