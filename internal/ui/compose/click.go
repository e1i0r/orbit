package compose

// One press of the mouse on the form: which field it landed in, which pill,
// or which of the three buttons at the foot.

import (
	"github.com/e1i0r/orbit/internal/ui/clip"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/typing"
)

// Click is one press of the mouse on the form.
func (s State) Click(t point.Target, e Env) (State, Out) {
	switch t.Kind {
	case point.ComposeTab:
		s.tab = t.Pane
		s.field = firstComposeField(t.Pane)

		return s, Out{}
	case point.ComposeFlowChoice:
		if t.Pane >= 0 && t.Pane < len(s.flows) {
			if s.flowIdx == t.Pane {
				return s, Out{Flow: s.flows[t.Pane]}
			}

			s.flowIdx = t.Pane
		}

		return s, Out{}
	case point.ComposeNewFlow:
		return s, Out{Flow: New}
	case point.ComposeInspectFlow:
		return s, Out{Flow: s.chosenFlow()}
	case point.ComposeField:
		s.field = t.Pane
		return s, Out{}
	case point.ComposeCaret:
		return s.Aim(t, e), Out{}
	case point.ComposeAction:
		switch t.Key {
		case "save":
			return s.Submit(false, e)
		case "save_and_run":
			return s.Submit(true, e)
		case "cancel":
			return State{}, Out{Leave: true}
		}
	case point.ComposePaste:
		if pasted := clip.Read(); pasted != "" {
			return s.Type(pasted), Out{}
		}
	}

	return s, Out{}
}

// Hit is what the form has at that cell.
func (s State) Hit(x, y int, e Env) point.Target {
	line, ok := e.Frame.BodyRow(y)
	if !ok {
		return point.Target{}
	}

	plan := s.composeLayout(e)

	if line == plan.tabLine {
		if x < 20 {
			return point.Target{Kind: point.ComposeTab, Pane: composeTabManual}
		}

		return point.Target{Kind: point.ComposeTab, Pane: composeTabURL}
	}

	// Which fields those rows are depends on the tab, and where they are
	// does not: both tabs are a flow, what that flow will do, and a box.
	flowField, boxField := composeFlow, composeText
	if s.tab == composeTabURL {
		flowField, boxField = composeURLFlow, composeURL
	}

	switch {
	case line == plan.flow:
		return s.hitComposeFlowPills(x, flowField, e)
	case plan.flowSum != -1 && line >= plan.flowSum && line < plan.flowSum+plan.flowRows:
		return point.Target{Kind: point.ComposeInspectFlow}
	case s.tab == composeTabManual && line == plan.id:
		return caretAt(composeID, 0, x-composeLabelStart)
	// The paste button ends the row the label and the top border are on,
	// so that row answers for it before it answers for the field.
	case line == plan.boxTop && s.onComposePaste(x, e):
		return point.Target{Kind: point.ComposePaste}
	case line == plan.boxTop, line == plan.boxBot:
		return point.Target{Kind: point.ComposeField, Pane: boxField}
	case line > plan.boxTop && line < plan.boxBot:
		return caretAt(boxField, line-plan.boxTop-1, x-composeBoxStart)
	case line >= plan.actions:
		return hitComposeActions(e.Words, x)
	}

	return point.Target{}
}

// Drag is the pointer moving with the button still down: the caret follows
// it while the anchor stays where the button went down, which is a selection
// being dragged out of the text.
func (s State) Drag(t point.Target, e Env) State {
	return s.composeExtend(func(mm State) State { return mm.Aim(t, e) })
}

// Aim is a cell of the form as a place inside a field: which field
// was pointed at, and where in what it holds.
func (s State) Aim(t point.Target, e Env) State {
	s.field = t.Pane

	if s.composeInBox() {
		return s.composePoint(t.Phase, t.Caret, e)
	}

	return s.composeCaret(func(in *typing.Field) { in.MoveTo(t.Caret) })
}
