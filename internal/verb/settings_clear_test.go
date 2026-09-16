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
		BudgetTask: 9, BudgetWorkspace: 99, QuotaFloor: 15,
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
