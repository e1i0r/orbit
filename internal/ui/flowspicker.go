package ui

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
func (m Model) openPicker(field int) Model {
	ids, _ := m.pickerChoices(field)

	at := 0

	for i, id := range ids {
		if id == m.pickedNow(field) {
			at = i
			break
		}
	}

	m.flows.picker = pickerState{open: true, field: field, sel: at}

	return m
}

// pickerChoices is what the field being picked can hold.
func (m Model) pickerChoices(field int) (ids, labels []string) {
	eng := m.dialEngine(m.flows.edited().Engine)

	switch field {
	case flowFieldSayEngine:
		names := m.engineNames()
		return names, names
	case flowFieldSayModel:
		mdls, mdlLabels := m.modelsFor(m.sayEngineName())
		return withDefault(m, mdls, mdlLabels)
	case flowFieldEngine:
		names := m.engineNames()
		return names, names
	case flowFieldModel:
		mdls, mdlLabels := m.modelsFor(eng)
		return withDefault(m, mdls, mdlLabels)
	case flowFieldEffort:
		effs, effLabels := m.effortsFor(eng)
		return withDefault(m, effs, effLabels)
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
func withDefault(m Model, ids, labels []string) (allIDs, allLabels []string) {
	fresh := m.opts.Words.T("flows.pick_default", "default (what the engine is set to)")

	return append([]string{""}, ids...), append([]string{fresh}, labels...)
}

// pickedNow is what that field holds at the moment.
func (m Model) pickedNow(field int) string {
	switch field {
	case flowFieldSayEngine:
		return m.sayEngineName()
	case flowFieldSayModel:
		return m.flows.sayModel
	case flowFieldEngine:
		return m.dialEngine(m.flows.edited().Engine)
	case flowFieldModel:
		return m.flows.edited().Model
	case flowFieldEffort:
		return m.flows.edited().Effort
	}

	return ""
}

// pickerRows is the choices that match what has been typed, as ids and the
// labels beside them.
func (m Model) pickerRows() (ids, labels []string) {
	all, allLabels := m.pickerChoices(m.flows.picker.field)

	want := strings.ToLower(m.flows.picker.filter)

	for i, id := range all {
		label := dialLabel(all, allLabels, i)
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
func (m Model) takePick(at int) Model {
	ids, _ := m.pickerRows()
	if at < 0 || at >= len(ids) {
		m.flows.picker = pickerState{}

		return m
	}

	switch m.flows.picker.field {
	case flowFieldSayEngine:
		// The model is one engine's own name for it, so it goes with the
		// engine it belonged to.
		m.flows.sayEngine, m.flows.sayModel = ids[at], ""
	case flowFieldSayModel:
		m.flows.sayModel = ids[at]
	case flowFieldEngine:
		// The model and the effort are one engine's own: kept across a
		// change of engine, they name something the new one has never
		// heard of, and internal/task refuses the phase before it runs.
		m.flows.edited().Engine = ids[at]
		m.flows.edited().Model = ""
		m.flows.edited().Effort = ""
	case flowFieldModel:
		m.flows.edited().Model = ids[at]
	case flowFieldEffort:
		m.flows.edited().Effort = ids[at]
	}

	m.flows.picker = pickerState{}

	return m
}

// pickerKey is every key while the list is up.
func (m Model) pickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	ids, _ := m.pickerRows()

	switch msg.Code {
	case tea.KeyEscape:
		m.flows.picker = pickerState{}
		return m, nil
	case tea.KeyEnter:
		return m.takePick(m.flows.picker.sel), nil
	case tea.KeyUp:
		m.flows.picker.sel = max(m.flows.picker.sel-1, 0)
		return m, nil
	case tea.KeyDown:
		m.flows.picker.sel = min(m.flows.picker.sel+1, max(len(ids)-1, 0))
		return m, nil
	case tea.KeyBackspace:
		m.flows.picker.filter = trimLastRune(m.flows.picker.filter)
		m.flows.picker.sel = 0

		return m, nil
	}

	if msg.Text != "" {
		m.flows.picker.filter += msg.Text
		m.flows.picker.sel = 0
	}

	return m, nil
}
