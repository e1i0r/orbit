package flows

// The list that picks a model, an engine or an effort.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
)

// TestThePickerOffersEveryChoiceTheEngineHas, which is the whole point of
// it: the row dial shows five at a time, and opencode has sixty.
func TestThePickerOffersEveryChoiceTheEngineHas(t *testing.T) {
	s, e := editing(t)
	s.field = flowFieldModel

	opened, _ := s.handleFlowFieldAction(e)

	s = opened
	if !s.picker.open {
		t.Fatal("enter on the model field did not open the list")
	}

	all, _ := e.Models(cells.OrDef(s.edited().Engine, e.Engine))

	// Every model the engine has, and the default above them.
	shown, _ := s.pickerRows(e)
	if len(shown) != len(all)+1 {
		t.Errorf("the list offers %d rows for the engine's %d models", len(shown), len(all))
	}
}

// TestTypingNarrowsTheList, because a reader looking for gpt-5.4-pro should
// not have to walk past sixty rows to reach it.
func TestTypingNarrowsTheList(t *testing.T) {
	s, e := editing(t)
	s = s.openPicker(flowFieldModel, e)

	all, _ := s.pickerRows(e)
	if len(all) < 2 {
		t.Skip("this build's engine table has too few models to narrow")
	}

	// The filter is a word out of the last row's own id, so what it must
	// keep is known and what it must drop is everything unlike it.
	last := all[len(all)-1]

	next, _ := s.pickerKey(tea.KeyPressMsg{Text: last}, e)
	after := next

	shown, _ := after.pickerRows(e)
	if len(shown) != 1 || shown[0] != last {
		t.Errorf("typing %q left %v, want just %q", last, shown, last)
	}

	// And what it narrowed to is what enter then takes.
	after = after.takePick(after.picker.sel, e)
	if got := after.edited().Model; got != last {
		t.Errorf("the narrowed list chose %q, want %q", got, last)
	}
}

// TestPickingWritesItIntoThePhase.
func TestPickingWritesItIntoThePhase(t *testing.T) {
	s, e := editing(t)
	s = s.openPicker(flowFieldModel, e)

	ids, _ := s.pickerRows(e)
	if len(ids) == 0 {
		t.Skip("this build's engine table has no models")
	}

	s = s.takePick(len(ids)-1, e)

	if got := s.edited().Model; got != ids[len(ids)-1] {
		t.Errorf("the phase's model is %q, want %q", got, ids[len(ids)-1])
	}

	if s.picker.open {
		t.Error("the list stayed up after a choice was made")
	}
}

// TestChangingEngineForgetsWhatWasTheOldOnesAlone. A model is one engine's
// own name for it, and internal/task refuses a phase whose model its engine
// has never heard of.
func TestChangingEngineForgetsWhatWasTheOldOnesAlone(t *testing.T) {
	s, e := editing(t)
	s.edited().Model = "opus"
	s.edited().Effort = "high"

	s = s.openPicker(flowFieldEngine, e)

	ids, _ := s.pickerRows(e)
	if len(ids) < 2 {
		t.Skip("this build has one engine")
	}

	s = s.takePick(1, e)
	if s.edited().Model != "" || s.edited().Effort != "" {
		t.Errorf("the old engine's model and effort survived the change: %+v", *s.edited())
	}
}

// TestClickingARowPicksIt is the mouse half: the rows are hit-tested from
// what was drawn, so a click on the fourth row chooses the fourth choice.
func TestClickingARowPicksIt(t *testing.T) {
	s, e := editing(t)
	s = s.openPicker(flowFieldModel, e)

	ids, _ := s.pickerRows(e)
	if len(ids) == 0 {
		t.Skip("this build's engine table has no models")
	}

	lines, _ := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)

	want := len(ids) - 1

	y := 0

	for i, l := range lines {
		if l.pick == want {
			y = e.Frame.Body.Y + i
		}
	}

	got := s.Hit(6, y, e)
	if got.Field != "pick" || got.Phase != want {
		t.Fatalf("hitFlows on the last row = %+v, want pick %d", got, want)
	}

	next, _ := s.Click(got, e)

	after := next
	if model := after.edited().Model; model != ids[want] {
		t.Errorf("clicking the last row chose %q, want %q", model, ids[want])
	}
}

// TestShiftEnterIsANewLine, in the two fields that hold a paragraph.
func TestShiftEnterIsANewLine(t *testing.T) {
	s, e := editing(t)
	s.field = flowFieldPrompt

	next, _ := s.flowsFormKey(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}, e)

	s = next
	if !strings.Contains(s.edited().Prompt, "\n") {
		t.Errorf("shift+enter left the instructions at %q", s.edited().Prompt)
	}

	// And on a one-line field it does nothing, rather than writing a
	// newline into a name.
	s.field = flowFieldName
	s.flowName = "mine"

	next, _ = s.flowsFormKey(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}, e)

	after := next
	if after.flowName != "mine" {
		t.Errorf("shift+enter on the name field left it as %q", after.flowName)
	}
}

// TestTheListCanPutAPhaseBackOnTheDefault, which the row of pills cannot:
// the empty value has no pill, so a phase moved off the default was stuck
// naming a model for the rest of its life.
func TestTheListCanPutAPhaseBackOnTheDefault(t *testing.T) {
	s, e := editing(t)
	s.edited().Model = "opus"

	s = s.openPicker(flowFieldModel, e)

	ids, labels := s.pickerRows(e)
	if len(ids) == 0 || ids[0] != "" {
		t.Fatalf("the list does not open with the default: %v / %v", ids, labels)
	}

	s = s.takePick(0, e)
	if got := s.edited().Model; got != "" {
		t.Errorf("choosing the default left the model at %q", got)
	}
}

// TestTheFormScrollsUnderTheWheel, because it is taller than a short
// terminal and the reader has to be able to see the end of it without
// leaving the field they are typing into.
func TestTheFormScrollsUnderTheWheel(t *testing.T) {
	s, e := designer(t)
	s = s.startCreateFlow(e).OnFields()

	lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)
	if len(lines) <= e.Frame.Body.H {
		t.Skip("the form fits this window, so there is nothing to scroll")
	}

	if start != 0 {
		t.Fatalf("a fresh form starts at row %d", start)
	}

	s = s.ScrollBuilder(notch, e)

	if _, after := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e); after != notch {
		t.Errorf("the wheel moved the window to %d, want %d", after, notch)
	}

	// It stops at the end rather than scrolling the form off the screen.
	for range 50 {
		s = s.ScrollBuilder(notch, e)
	}

	lines, end := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)
	if end != len(lines)-e.Frame.Body.H {
		t.Errorf("the wheel scrolled to %d of %d rows", end, len(lines))
	}

	// And moving to a field brings the window back onto it.
	s.field = flowFieldTemplate
	s = s.followField(e)

	lines, back := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)

	at := s.fieldRow(lines)
	if at < back || at >= back+e.Frame.Body.H {
		t.Errorf("the first field is at row %d, outside the window at %d", at, back)
	}
}
