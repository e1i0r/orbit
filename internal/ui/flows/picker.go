package flows

// Picking one value out of a list too long to walk.
//
// opencode answers to sixty-odd models. A dial you press right on sixty
// times is a dial nobody uses, so the model, the effort and the engine open
// a list instead: every choice the engine has, one per row, with a filter
// over it and the mouse able to land on any of them. The row dial stays for
// the short lists, and for seeing what is set without opening anything.

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
)

// pickerState is the list while it is up: which field it is choosing for,
// where the cursor is in it, and what has been typed to narrow it.
type pickerState struct {
	open   bool
	field  int
	sel    int
	filter string
}

// openPicker raises the list for the field the reader is on, with the cursor
// on what is already chosen — the thing they are most likely to be looking
// for is the neighbours of what they have.
func (s State) openPicker(field int, e Env) State {
	ids, _ := s.pickerChoices(field, e)

	at := 0

	for i, id := range ids {
		if id == s.pickedNow(field, e) {
			at = i
			break
		}
	}

	s.picker = pickerState{open: true, field: field, sel: at}

	return s
}

// pickerChoices is what the field being picked can hold.
func (s State) pickerChoices(field int, e Env) (ids, labels []string) {
	eng := cells.OrDef(s.edited().Engine, e.Engine)

	switch field {
	case flowFieldSayEngine:
		names := e.Engines()
		return names, names
	case flowFieldSayModel:
		mdls, mdlLabels := e.Models(s.sayEngineName(e))
		return withDefault(mdls, mdlLabels, e)
	case flowFieldEngine:
		names := e.Engines()
		return names, names
	case flowFieldModel:
		mdls, mdlLabels := e.Models(eng)
		return withDefault(mdls, mdlLabels, e)
	case flowFieldEffort:
		effs, effLabels := e.Efforts(eng)
		return withDefault(effs, effLabels, e)
	}

	return nil, nil
}

// withDefault puts "whatever the engine is set to" at the top of the list.
//
// The row dial cannot offer it — a click there is routed by the value it
// drew, and an empty one is indistinguishable from a click on no pill at all
// — so a phase moved off the default could never be moved back. Here the
// rows are routed by their position, so the empty value has a row like any
// other.
func withDefault(ids, labels []string, e Env) (allIDs, allLabels []string) {
	fresh := e.Words.T("flows.pick_default", "default")

	return append([]string{""}, ids...), append([]string{fresh}, labels...)
}

// pickedNow is what that field holds at the moment.
func (s State) pickedNow(field int, e Env) string {
	switch field {
	case flowFieldSayEngine:
		return s.sayEngineName(e)
	case flowFieldSayModel:
		return s.sayModel
	case flowFieldEngine:
		return cells.OrDef(s.edited().Engine, e.Engine)
	case flowFieldModel:
		return s.edited().Model
	case flowFieldEffort:
		return s.edited().Effort
	}

	return ""
}

// pickerRows is the choices that match what has been typed, as ids and the
// labels beside them.
func (s State) pickerRows(e Env) (ids, labels []string) {
	all, allLabels := s.pickerChoices(s.picker.field, e)

	want := strings.ToLower(s.picker.filter)

	for i, id := range all {
		label := cells.DialLabel(all, allLabels, i)
		if want != "" && !strings.Contains(strings.ToLower(id+" "+label), want) {
			continue
		}

		ids = append(ids, id)
		labels = append(labels, label)
	}

	return ids, labels
}

// takePick writes the choice under the cursor into the phase and closes the
// list.
func (s State) takePick(at int, e Env) State {
	ids, _ := s.pickerRows(e)
	if at < 0 || at >= len(ids) {
		s.picker = pickerState{}

		return s
	}

	switch s.picker.field {
	case flowFieldSayEngine:
		// The model is one engine's own name for it, so it goes with the
		// engine it belonged to.
		s.sayEngine, s.sayModel = ids[at], ""
	case flowFieldSayModel:
		s.sayModel = ids[at]
	case flowFieldEngine:
		// The model and the effort are one engine's own: kept across a
		// change of engine, they name something the new one has never
		// heard of, and internal/task refuses the phase before it runs.
		s.edited().Engine = ids[at]
		s.edited().Model = ""
		s.edited().Effort = ""
	case flowFieldModel:
		s.edited().Model = ids[at]
	case flowFieldEffort:
		s.edited().Effort = ids[at]
	}

	s.picker = pickerState{}

	return s
}

// pickerKey is every key while the list is up.
func (s State) pickerKey(msg tea.KeyPressMsg, e Env) (State, Out) {
	ids, _ := s.pickerRows(e)

	switch msg.Code {
	case tea.KeyEscape:
		s.picker = pickerState{}
		return s, Out{}
	case tea.KeyEnter:
		return s.takePick(s.picker.sel, e), Out{}
	case tea.KeyUp:
		s.picker.sel = max(s.picker.sel-1, 0)
		return s, Out{}
	case tea.KeyDown:
		s.picker.sel = min(s.picker.sel+1, max(len(ids)-1, 0))
		return s, Out{}
	case tea.KeyBackspace:
		s.picker.filter = cells.TrimLastRune(s.picker.filter)
		s.picker.sel = 0

		return s, Out{}
	}

	if msg.Text != "" {
		s.picker.filter += msg.Text
		s.picker.sel = 0
	}

	return s, Out{}
}
