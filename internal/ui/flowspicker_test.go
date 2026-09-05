package ui

// The list that picks a model, an engine or an effort.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestThePickerOffersEveryChoiceTheEngineHas, which is the whole point of
// it: the row dial shows five at a time, and opencode has sixty.
func TestThePickerOffersEveryChoiceTheEngineHas(t *testing.T) {
	m := builderModel(t)
	m.flows.field = flowFieldModel

	opened, _ := m.handleFlowFieldAction()

	m = asModel(t, opened)
	if !m.flows.picker.open {
		t.Fatal("enter on the model field did not open the list")
	}

	all, _ := m.modelsFor(m.dialEngine(m.flows.edited().Engine))

	// Every model the engine has, and the default above them.
	shown, _ := m.pickerRows()
	if len(shown) != len(all)+1 {
		t.Errorf("the list offers %d rows for the engine's %d models", len(shown), len(all))
	}
}

// TestTypingNarrowsTheList, because a reader looking for gpt-5.4-pro should
// not have to walk past sixty rows to reach it.
func TestTypingNarrowsTheList(t *testing.T) {
	m := builderModel(t)
	m = m.openPicker(flowFieldModel)

	all, _ := m.pickerRows()
	if len(all) < 2 {
		t.Skip("this build's engine table has too few models to narrow")
	}

	// The filter is a word out of the last row's own id, so what it must
	// keep is known and what it must drop is everything unlike it.
	last := all[len(all)-1]

	next, _ := m.pickerKey(tea.KeyPressMsg{Text: last})
	after := asModel(t, next)

	shown, _ := after.pickerRows()
	if len(shown) != 1 || shown[0] != last {
		t.Errorf("typing %q left %v, want just %q", last, shown, last)
	}

	// And what it narrowed to is what enter then takes.
	after = after.takePick(after.flows.picker.sel)
	if got := after.flows.edited().Model; got != last {
		t.Errorf("the narrowed list chose %q, want %q", got, last)
	}
}

// TestPickingWritesItIntoThePhase.
func TestPickingWritesItIntoThePhase(t *testing.T) {
	m := builderModel(t)
	m = m.openPicker(flowFieldModel)

	ids, _ := m.pickerRows()
	if len(ids) == 0 {
		t.Skip("this build's engine table has no models")
	}

	m = m.takePick(len(ids) - 1)

	if got := m.flows.edited().Model; got != ids[len(ids)-1] {
		t.Errorf("the phase's model is %q, want %q", got, ids[len(ids)-1])
	}

	if m.flows.picker.open {
		t.Error("the list stayed up after a choice was made")
	}
}

// TestChangingEngineForgetsWhatWasTheOldOnesAlone. A model is one engine's
// own name for it, and internal/task refuses a phase whose model its engine
// has never heard of.
func TestChangingEngineForgetsWhatWasTheOldOnesAlone(t *testing.T) {
	m := builderModel(t)
	m.flows.edited().Model = "opus"
	m.flows.edited().Effort = "high"

	m = m.openPicker(flowFieldEngine)

	ids, _ := m.pickerRows()
	if len(ids) < 2 {
		t.Skip("this build has one engine")
	}

	m = m.takePick(1)
	if m.flows.edited().Model != "" || m.flows.edited().Effort != "" {
		t.Errorf("the old engine's model and effort survived the change: %+v", *m.flows.edited())
	}
}

// TestClickingARowPicksIt is the mouse half: the rows are hit-tested from
// what was drawn, so a click on the fourth row chooses the fourth choice.
func TestClickingARowPicksIt(t *testing.T) {
	m := builderModel(t)
	m = m.openPicker(flowFieldModel)

	ids, _ := m.pickerRows()
	if len(ids) == 0 {
		t.Skip("this build's engine table has no models")
	}

	lines, _ := m.builderView(m.frame.Body.H, m.frame.Body.W)

	want := len(ids) - 1

	y := 0

	for i, l := range lines {
		if l.pick == want {
			y = m.frame.Body.Y + i
		}
	}

	got := m.hitFlows(6, y)
	if got.Field != "pick" || got.Phase != want {
		t.Fatalf("hitFlows on the last row = %+v, want pick %d", got, want)
	}

	next, _ := m.handleFlowClick(got)

	after := asModel(t, next)
	if model := after.flows.edited().Model; model != ids[want] {
		t.Errorf("clicking the last row chose %q, want %q", model, ids[want])
	}
}

// TestShiftEnterIsANewLine, in the two fields that hold a paragraph.
func TestShiftEnterIsANewLine(t *testing.T) {
	m := builderModel(t)
	m.flows.field = flowFieldPrompt

	next, _ := m.flowsFormKey(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift})

	m = asModel(t, next)
	if !strings.Contains(m.flows.edited().Prompt, "\n") {
		t.Errorf("shift+enter left the instructions at %q", m.flows.edited().Prompt)
	}

	// And on a one-line field it does nothing, rather than writing a
	// newline into a name.
	m.flows.field = flowFieldName
	m.flows.flowName = "mine"

	next, _ = m.flowsFormKey(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift})

	after := asModel(t, next)
	if after.flows.flowName != "mine" {
		t.Errorf("shift+enter on the name field left it as %q", after.flows.flowName)
	}
}

// TestTheListCanPutAPhaseBackOnTheDefault, which the row of pills cannot:
// the empty value has no pill, so a phase moved off the default was stuck
// naming a model for the rest of its life.
func TestTheListCanPutAPhaseBackOnTheDefault(t *testing.T) {
	m := builderModel(t)
	m.flows.edited().Model = "opus"

	m = m.openPicker(flowFieldModel)

	ids, labels := m.pickerRows()
	if len(ids) == 0 || ids[0] != "" {
		t.Fatalf("the list does not open with the default: %v / %v", ids, labels)
	}

	m = m.takePick(0)
	if got := m.flows.edited().Model; got != "" {
		t.Errorf("choosing the default left the model at %q", got)
	}
}
