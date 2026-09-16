package known

// What the screen does with the store it was handed: reading it once for the
// header, saying so when there is none, and refusing what it cannot write.

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// TestTheStoreIsReadOnceForTheHeadersChip. Reading walks every repository on
// the board, and the count only changes when a fact is written — which is
// where the store is read again.
func TestTheStoreIsReadOnceForTheHeadersChip(t *testing.T) {
	reads := 0

	e := world(t)
	e.All = func() []knowledge.Rule {
		reads++

		return []knowledge.Rule{
			known("of everything", knowledge.Scope{Kind: knowledge.General}),
			known("of Go", knowledge.Scope{Kind: knowledge.Language, Lang: "go"}),
		}
	}

	s := State{}.SyncOnce(e)

	if reads != 1 {
		t.Fatalf("the first board read the store %d times", reads)
	}

	if got := s.Count(); got != 2 {
		t.Errorf("the chip says %d facts, want the two the store holds", got)
	}

	if s = s.SyncOnce(e); reads != 1 {
		t.Errorf("the second board read the store again (%d reads)", reads)
	}

	// Writing one is where it is read again.
	if _ = s.Sync(e); reads != 2 {
		t.Errorf("a fact written did not read the store again (%d reads)", reads)
	}
}

// TestAWindowWithNoStoreSaysNothingRatherThanReadingNothing.
func TestAWindowWithNoStoreSaysNothingRatherThanReadingNothing(t *testing.T) {
	e := world(t)
	e.All = nil

	s := Open(e)
	if got := s.Count(); got != 0 {
		t.Errorf("a window with no store counted %d facts", got)
	}

	if drawn := drawnKnowledge(t, s, e); !strings.Contains(drawn, "Nothing written down") {
		t.Errorf("a window with no store drew:\n%s", drawn)
	}
}

// TestTheKeysThatDoNothingHereDoNothing. A reader on this screen should not
// be able to move a cursor on the one behind it.
func TestTheKeysThatDoNothingHereDoNothing(t *testing.T) {
	s, e := onScreen(t, known("of everything", knowledge.Scope{Kind: knowledge.General}))

	after, out := s.Key(press("z"), e)
	if after.sel != s.sel || out.Leave || out.Said != "" {
		t.Errorf("a key the screen has no use for answered %+v", out)
	}
}

// TestDecidingAgainstARuleIsRefusedWhenThereIsNowhereToWriteIt, rather than
// leaving the screen showing a decision nothing recorded.
func TestDecidingAgainstARuleIsRefusedWhenThereIsNowhereToWriteIt(t *testing.T) {
	s, e := onScreen(t, known("of everything", knowledge.Scope{Kind: knowledge.General}))
	e.Replace = nil

	s, _ = s.Key(press("enter"), e)

	if after, out := s.Key(press("o"), e); after.facts[0].State != knowledge.Active || out.Said != "" {
		t.Errorf("deciding against it with no door to write through answered %+v", out)
	}

	// And the store's own refusal is said in the store's words.
	e.Replace = func(_, _ knowledge.Rule) error { return errors.New("the store is read-only") }

	if _, out := s.Key(press("o"), e); !strings.Contains(out.Said, "read-only") {
		t.Errorf("the refusal reads %q", out.Said)
	}
}

// TestEditingIsRefusedWhenThereIsNowhereToSaveIt. Typing into a line that
// cannot be written back is worse than not offering it.
func TestEditingIsRefusedWhenThereIsNowhereToSaveIt(t *testing.T) {
	s, e := onScreen(t, known("of everything", knowledge.Scope{Kind: knowledge.General}))
	e.Replace = nil

	if after, _ := s.Key(press("enter"), e); after.editing {
		t.Error("enter opened a form with nowhere to save it to")
	}

	if after, _ := s.Key(press("n"), e); after.editing {
		t.Error("n opened a line with nowhere to save it to")
	}
}

// TestAFactSaysHowMuchUseItHasHadAndWhereItCameFrom. A sentence in the
// agent's context that nobody can trace is indistinguishable from one the
// model made up.
func TestAFactSaysHowMuchUseItHasHadAndWhereItCameFrom(t *testing.T) {
	told := known("the api refuses a body over 1MB", knowledge.Scope{Kind: knowledge.General})
	told.Source, told.Ref, told.Used = knowledge.FromCode, "internal/api/limits.go", 12
	told.At = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	s, e := onScreen(t, told)

	// On the rule's own screen. The list has one line a rule and spends it
	// on the sentence; the question "can I trust this" is asked of one rule
	// at a time, and this is where it is answered.
	drawn := ansi.Strip(strings.Join(s.openDetail(e).View(30, 96, e), "\n"))
	for _, want := range []string{"the code", "2026-09-01", "HITS"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the rule does not say %q:\n%s", want, drawn)
		}
	}
}

// TestARuleWithACheckSaysItStops, and one without says it cannot fire yet:
// the difference is the whole point of the two categories.
func TestARuleWithACheckSaysItStops(t *testing.T) {
	armed := known("coverage stays above 90%", knowledge.Scope{Kind: knowledge.General})
	armed.Stops, armed.Check = true, "make coverage"

	s, e := onScreen(t, armed)
	if drawn := drawnKnowledge(t, s, e); !strings.Contains(drawn, "⚡ BLOCKED") {
		t.Errorf("a rule with a check is not under what stops the work:\n%s", drawn)
	}

	off := armed
	off.State = knowledge.Off

	s, e = onScreen(t, off)
	if drawn := drawnKnowledge(t, s, e); !strings.Contains(drawn, "🚫 TURNED OFF") {
		t.Errorf("a rule that is off is not under OFF:\n%s", drawn)
	}
}
