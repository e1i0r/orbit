package flows

// The list a long dial opens instead of turning: opencode answers to sixty
// models, and a dial pressed sixty times is a dial nobody uses.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// picking is the builder with the list up on one field.
func picking(t *testing.T, field int) (State, Env) {
	t.Helper()

	s, e := editing(t)
	s = s.openPicker(field, e)

	if !s.picker.open {
		t.Fatalf("the list did not open on field %d", field)
	}

	return s, e
}

// TestTheListOpensOnWhatIsAlreadyChosen. The thing a reader is looking for
// is usually the neighbours of what they have.
func TestTheListOpensOnWhatIsAlreadyChosen(t *testing.T) {
	s, e := editing(t)

	ids, _ := s.pickerChoices(flowFieldModel, e)
	if len(ids) < 3 {
		t.Fatalf("the model list offers %d choices, want the default and zeta's two", len(ids))
	}

	// Choose the last model, close, and open again: the cursor is on it.
	s.edited().Model = ids[len(ids)-1]

	s = s.openPicker(flowFieldModel, e)
	if s.picker.sel != len(ids)-1 {
		t.Errorf("the list opened on row %d, want the model already chosen at %d", s.picker.sel, len(ids)-1)
	}
}

// TestEveryDialThatOpensAListOffersItsOwnChoices.
func TestEveryDialThatOpensAListOffersItsOwnChoices(t *testing.T) {
	s, e := editing(t)

	for _, c := range []struct {
		field int
		want  string
	}{
		{flowFieldEngine, "zeta"},
		{flowFieldModel, "zeta/one"},
		{flowFieldEffort, "brisk"},
		{flowFieldSayEngine, "zeta"},
		{flowFieldSayModel, "zeta/one"},
	} {
		ids, labels := s.pickerChoices(c.field, e)
		if len(ids) != len(labels) {
			t.Errorf("field %d offers %d ids and %d labels", c.field, len(ids), len(labels))
		}

		if !strings.Contains(strings.Join(ids, " "), c.want) {
			t.Errorf("field %d offers %v, want it to include %q", c.field, ids, c.want)
		}
	}

	// A field that opens no list has nothing to offer.
	if ids, _ := s.pickerChoices(flowFieldName, e); len(ids) != 0 {
		t.Errorf("a field with no list offered %v", ids)
	}
}

// TestTypingNarrowsTheListAndBackspaceGivesItBack.
func TestTypingNarrowsTheListAndBackspaceGivesItBack(t *testing.T) {
	s, e := picking(t, flowFieldModel)

	all, _ := s.pickerRows(e)

	for _, r := range "two" {
		s, _ = s.pickerKey(press(string(r)), e)
	}

	few, _ := s.pickerRows(e)
	if len(few) >= len(all) || len(few) == 0 {
		t.Fatalf("typing left %d of %d choices", len(few), len(all))
	}

	for _, id := range few {
		if !strings.Contains(strings.ToLower(id), "two") {
			t.Errorf("%q is on the filtered list", id)
		}
	}

	for range 3 {
		s, _ = s.pickerKey(tea.KeyPressMsg{Code: tea.KeyBackspace}, e)
	}

	if back, _ := s.pickerRows(e); len(back) != len(all) {
		t.Errorf("taking the filter back off left %d choices, want all %d", len(back), len(all))
	}
}

// TestTheCursorStopsAtBothEndsOfTheList: a list that comes back round has no
// bottom, and somebody holding a key down never finds out they reached it.
func TestTheCursorStopsAtBothEndsOfTheList(t *testing.T) {
	s, e := picking(t, flowFieldModel)
	s.picker.sel = 0

	if up, _ := s.pickerKey(press("up"), e); up.picker.sel != 0 {
		t.Errorf("up from the first row left the cursor on %d", up.picker.sel)
	}

	ids, _ := s.pickerRows(e)
	s.picker.sel = len(ids) - 1

	if down, _ := s.pickerKey(press("down"), e); down.picker.sel != len(ids)-1 {
		t.Errorf("down from the last row left the cursor on %d", down.picker.sel)
	}
}

// TestChoosingAnEngineDropsTheModelAndEffortThatBelongedToTheOther. Kept
// across a change of engine they name something the new one has never heard
// of, and internal/task refuses the phase before it runs.
func TestChoosingAnEngineDropsTheModelAndEffortThatBelongedToTheOther(t *testing.T) {
	s, e := picking(t, flowFieldEngine)
	s.edited().Model, s.edited().Effort = "zeta/two", "brisk"

	s = s.takePick(0, e)

	if s.edited().Model != "" || s.edited().Effort != "" {
		t.Errorf("changing the engine kept model=%q effort=%q",
			s.edited().Model, s.edited().Effort)
	}

	if s.picker.open {
		t.Error("the list stayed up after a choice")
	}
}

// TestChoosingTheSayEngineDropsItsModelForTheSameReason.
func TestChoosingTheSayEngineDropsItsModelForTheSameReason(t *testing.T) {
	s, e := picking(t, flowFieldSayModel)

	s = s.takePick(1, e)
	if s.sayModel == "" {
		t.Error("choosing a model for the draft left it unset")
	}

	s = s.openPicker(flowFieldSayEngine, e)
	s = s.takePick(0, e)

	if s.sayModel != "" {
		t.Errorf("changing the engine the draft is asked of kept model %q", s.sayModel)
	}
}

// TestAChoiceOffTheEndOfTheListChangesNothingAndClosesIt, which is what a
// click on a row the list no longer has is.
func TestAChoiceOffTheEndOfTheListChangesNothingAndClosesIt(t *testing.T) {
	s, e := picking(t, flowFieldModel)
	s.edited().Model = "zeta/one"

	after := s.takePick(99, e)
	if after.edited().Model != "zeta/one" {
		t.Errorf("a row nobody drew set the model to %q", after.edited().Model)
	}

	if after.picker.open {
		t.Error("the list stayed up after a choice off the end of it")
	}
}

// TestEscapeClosesTheListAndChangesNothing.
func TestEscapeClosesTheListAndChangesNothing(t *testing.T) {
	s, e := picking(t, flowFieldModel)
	s.edited().Model = "zeta/one"

	after, out := s.pickerKey(tea.KeyPressMsg{Code: tea.KeyEscape}, e)
	if after.picker.open || after.edited().Model != "zeta/one" {
		t.Errorf("escape left the list open=%v model=%q", after.picker.open, after.edited().Model)
	}

	if out.Said != "" {
		t.Errorf("escape said %q", out.Said)
	}
}

// TestTheListIsDrawnWithTheChoicesItHolds.
func TestTheListIsDrawnWithTheChoicesItHolds(t *testing.T) {
	s, e := picking(t, flowFieldModel)

	// The labels and not the ids: "zeta/one" is what the engine calls it
	// and "one" is what the reader picked it by.
	drawn := ansi.Strip(strings.Join(s.View(30, 100, e), "\n"))
	for _, want := range []string{"one", "two"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the list does not carry %q:\n%s", want, drawn)
		}
	}
}
