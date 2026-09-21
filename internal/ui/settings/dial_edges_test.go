package settings

// The ends of a dial, and the rows that hold nothing.
//
// Everything here is a reading the suite made about the middle of a range
// and never about its ends — which is where a dial whose engine offers one
// thing, or none, actually lands.

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestASettingThatHoldsNothingSaysSoInAWord. An empty space beside a name
// is a row a reader cannot tell from a broken one, and a line ending in "is
// back to" reads as a sentence that broke off.
func TestASettingThatHoldsNothingSaysSoInAWord(t *testing.T) {
	e := env(t, newFile())

	if got := shown(e, ""); got == "" {
		t.Error("a setting put back to nothing was said as nothing at all")
	}

	if got := shown(e, "brisk"); got != "brisk" {
		t.Errorf("a setting put back to brisk was said as %q", got)
	}

	if got := written(""); got == "" {
		t.Error("a row holding nothing drew nothing")
	}

	if got := written("zeta/one"); got != "zeta/one" {
		t.Errorf("a row holding zeta/one drew %q", got)
	}
}

// TestOnlyTheLanguageAsksTheWindowToReload. The window reloads its
// catalogue on the answer this carries, so a setting that carried it when it
// was not the language would rebuild every string on the screen for a
// change of theme.
func TestOnlyTheLanguageAsksTheWindowToReload(t *testing.T) {
	e := env(t, newFile())

	if out := Blank("language", e); out.Lang == "" {
		t.Error("the language put back did not ask the window to reload")
	}

	for _, name := range []string{"autopilot", "theme", "flow"} {
		if out := Blank(name, e); out.Lang != "" {
			t.Errorf("%s put back asked the window to reload in %q", name, out.Lang)
		}
	}

	if out := Apply("theme", "frauddi", e); out.Lang != "" {
		t.Errorf("a theme change asked the window to reload in %q", out.Lang)
	}
}

// TestAnEngineOfferingNothingChangesNothingUnderIt. The model and the
// effort follow the engine, and an engine this build has nothing to offer
// for is one where following it means reading the first of an empty list.
func TestAnEngineOfferingNothingChangesNothingUnderIt(t *testing.T) {
	f := newFile()
	e := env(t, f)
	e.Models = func(string) (ids, labels []string) { return nil, nil }
	e.Efforts = func(string) (ids, labels []string) { return nil, nil }

	out := Apply("engine", "omega", e)
	if out.Said == "" {
		t.Error("changing the engine said nothing")
	}

	if out.Dials != nil {
		t.Errorf("an engine with nothing under it moved the dials to %+v", out.Dials)
	}

	if got := f.held["engine"]; got != "omega" {
		t.Errorf("the file holds engine=%q, want omega", got)
	}
}

// TestTheModelIsReadOffTheTableOrIsNothing. It is read back off the table
// rather than through a getter of its own, so the two ways the table can
// fail to name one — no table at all, and a table without that row — are
// what stands between it and a model nobody has.
func TestTheModelIsReadOffTheTableOrIsNothing(t *testing.T) {
	e := env(t, newFile())
	if got := modelNow(e); got != "zeta/one" {
		t.Errorf("the model the file holds reads as %q", got)
	}

	none := e
	none.Kept = nil

	if got := modelNow(none); got != "" {
		t.Errorf("with no table at all the model reads as %q", got)
	}

	other := e
	other.Kept = func() []Kept { return []Kept{{Name: "engine", Value: "zeta"}} }

	if got := modelNow(other); got != "" {
		t.Errorf("a table with no model row reads its model as %q", got)
	}
}

// TestTheFlowDialOffersTheBuildsFlowsWhenTheReaderHasNone. Names written out
// by hand on this dial leave every flow they do not list impossible to
// choose, so the dial falls back to what the binary ships rather than to
// nothing.
func TestTheFlowDialOffersTheBuildsFlowsWhenTheReaderHasNone(t *testing.T) {
	e := env(t, newFile())

	var s State

	opts, _ := s.offers("flow", e)
	if len(opts) == 0 {
		t.Error("a screen that has read no flows offers none at all")
	}

	s.flows = []string{"cover", "quick"}

	if opts, _ := s.offers("flow", e); len(opts) != 2 {
		t.Errorf("a screen that read two flows offers %v", opts)
	}
}

// TestTheDialsAreReadAgainstTheEngineTheFileHolds, which is the row named
// engine and not whichever row happens to be first.
func TestTheDialsAreReadAgainstTheEngineTheFileHolds(t *testing.T) {
	e := env(t, newFile())

	var s State

	if got := s.engine(e); got != "zeta" {
		t.Errorf("the dials are read against %q, want the engine the file holds", got)
	}

	other := e
	other.Kept = func() []Kept { return []Kept{{Name: "model", Value: "zeta/one"}} }

	if got := s.engine(other); got != "zeta" {
		t.Errorf("with no engine row the dials are read against %q, want the first this build has", got)
	}
}

// TestALeftAndARightTurnTheDialOppositeWays, and each lands on the option
// beside the one held. A dial that moved the same way on both keys is one a
// reader cannot get back to where they started with, and one that moved two
// options at a time skips whatever is between them.
func TestALeftAndARightTurnTheDialOppositeWays(t *testing.T) {
	for _, c := range []struct {
		key  tea.KeyPressMsg
		want string
	}{
		{tea.KeyPressMsg{Code: tea.KeyRight}, "5"},
		{tea.KeyPressMsg{Code: 'l', Text: "l"}, "5"},
		{tea.KeyPressMsg{Code: tea.KeyLeft}, "0"},
		{tea.KeyPressMsg{Code: 'h', Text: "h"}, "0"},
	} {
		f := newFile()
		f.held["unread-cap"] = "3" // the middle of 0, 3, 5, 10, 20

		e := env(t, f)

		s := Open(e)
		s.sel = rowOf(t, s, e, "unread-cap")

		s.Key(c.key, e)

		if got := f.held["unread-cap"]; got != c.want {
			t.Errorf("%v on a dial holding 3 left it on %q, want %q", c.key, got, c.want)
		}
	}
}

// TestADialTurnedOffTheEndComesBackOnTheOther, either way round.
func TestADialTurnedOffTheEndComesBackOnTheOther(t *testing.T) {
	for _, c := range []struct {
		from  string
		delta int
		want  string
	}{
		{"20", 1, "0"},
		{"0", -1, "20"},
	} {
		f := newFile()
		f.held["unread-cap"] = c.from

		e := env(t, f)

		s := Open(e)
		s.sel = rowOf(t, s, e, "unread-cap")

		s.Cycle(c.delta, e)

		if got := f.held["unread-cap"]; got != c.want {
			t.Errorf("a dial on %q turned %+d is on %q, want %q", c.from, c.delta, got, c.want)
		}
	}
}

// TestARowOnePastTheLastIsNoRow. The cursor is an index into a table that
// can get shorter under it — the model and effort dials are the engine's —
// so every door that reads a row by index has an end on both sides, and the
// one that matters is the row after the last.
func TestARowOnePastTheLastIsNoRow(t *testing.T) {
	e := env(t, newFile())

	s := Open(e)
	n := len(s.Rows(e))

	for _, sel := range []int{-1, n, n + 4} {
		s.sel = sel

		if out := s.Cycle(1, e); out.Said != "" {
			t.Errorf("turning the dial of row %d of %d said %q", sel, n, out.Said)
		}

		open := s
		open.editing, open.typed = true, "algo"

		if next, out := open.Key(tea.KeyPressMsg{Code: tea.KeyEnter}, e); next.editing || out.Said != "" {
			t.Errorf("a line committed on row %d of %d answered %q", sel, n, out.Said)
		}
	}
}

// TestALineTypedIntoTheFirstRowIsWritten. The row a line is committed to is
// found by index, and the guard around that index has an end at either side:
// the first row is inside the table and a row one past the last is not.
func TestALineTypedIntoTheFirstRowIsWritten(t *testing.T) {
	f := newFile()
	e := env(t, f)

	s := Open(e)
	if s.Chosen() != 0 {
		t.Fatalf("the screen opens on row %d, want the first", s.Chosen())
	}

	first := s.Rows(e)[0]

	s, _ = s.Key(tea.KeyPressMsg{Code: 'e', Text: "e"}, e)
	if !s.Editing() {
		t.Fatal("e on the first row did not open a line to type in")
	}

	// The line opens on what the row holds, so what is committed is that
	// and whatever was typed after it.
	for _, r := range "es" {
		s, _ = s.Key(tea.KeyPressMsg{Code: r, Text: string(r)}, e)
	}

	s, out := s.Key(tea.KeyPressMsg{Code: tea.KeyEnter}, e)
	if s.Editing() {
		t.Error("⏎ left the line open")
	}

	if out.Said == "" {
		t.Error("a line committed on the first row said nothing")
	}

	if got, want := f.held[first.Key], first.Val+"es"; got != want {
		t.Errorf("the first row holds %q after a line was typed into it, want %q", got, want)
	}
}

// TestTheFirstRowIsARowLikeAnyOther. The cursor opens on it, so a guard
// that read row zero as no row at all would leave the first dial on the
// screen the one dial nothing turns.
func TestTheFirstRowIsARowLikeAnyOther(t *testing.T) {
	f := newFile()
	e := env(t, f)

	s := Open(e)
	if s.Chosen() != 0 {
		t.Fatalf("the screen opens on row %d, want the first", s.Chosen())
	}

	first := s.Rows(e)[0]

	if out := s.Cycle(1, e); out.Said == "" {
		t.Error("turning the first row's dial said nothing")
	}

	if got := f.held[first.Key]; got == first.Val {
		t.Errorf("the first row's dial is still on %q after being turned", got)
	}
}
