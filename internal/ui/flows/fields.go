package flows

// Which fields the designer shows, and what typing into one does.
//
// The list is asked for rather than counted, because it is no longer fixed:
// a phase that repeats shows two fields a phase that does not has nothing to
// say about. Tab, the mouse and the draw all read this one list, so a field
// that appears cannot appear in one of them and not the others.

import (
	"slices"

	"github.com/e1i0r/orbit/internal/ui/cells"
)

// fieldsShown is every field of the form, in the order it is drawn.
func (s *State) fieldsShown() []int {
	out := []int{
		flowFieldTemplate,
		flowFieldName,
		flowFieldDescription,
		flowFieldPhaseSelect,
		flowFieldPhaseName,
		flowFieldIsLoop,
	}

	if s.looping() {
		out = append(out, flowFieldLoopTurns, flowFieldLoopUntil)
	}

	return append(out,
		flowFieldEngine,
		flowFieldModel,
		flowFieldEffort,
		flowFieldThinking,
		flowFieldFeedOutput,
		flowFieldWait,
		flowFieldPrompt,
		flowFieldAddPhase,
		flowFieldDelPhase,
		flowFieldSave,
	)
}

// moveField is tab and shift-tab: the next field that is actually on screen.
//
// A field the form has stopped showing — the loop's two, after the switch
// went off — lands on the switch itself, which is where the reader was when
// it happened and what they would have to press to get the fields back.
func (s *State) moveField(delta int) {
	fields := s.fieldsShown()

	at := slices.Index(fields, s.field)
	if at < 0 {
		s.field = flowFieldIsLoop
		return
	}

	s.field = fields[(at+delta+len(fields))%len(fields)]
}

// typing is whether the field under the cursor takes characters, as opposed
// to a dial that left and right turn.
func (s *State) typing() bool {
	switch s.field {
	case flowFieldName, flowFieldDescription, flowFieldPhaseName,
		flowFieldPrompt, flowFieldLoopUntil:
		return true
	}

	return false
}

// multiline is whether a new line belongs in it.
//
// The purpose, the instructions and the list of checks are paragraphs. A
// flow's name is not: it is the name of the file the flow is written to, and
// a newline in it is a file name nothing can open.
func (s *State) multiline() bool {
	switch s.field {
	case flowFieldDescription, flowFieldPrompt, flowFieldLoopUntil:
		return true
	}

	return false
}

// rub takes the last character back out of it.
func (s *State) rub() {
	switch s.field {
	case flowFieldName:
		s.flowName = cells.TrimLastRune(s.flowName)
	case flowFieldDescription:
		s.description = cells.TrimLastRune(s.description)
	case flowFieldPhaseName:
		s.cur().Name = cells.TrimLastRune(s.cur().Name)
	case flowFieldPrompt:
		s.edited().Prompt = cells.TrimLastRune(s.edited().Prompt)
	case flowFieldLoopUntil:
		s.setChecks(cells.TrimLastRune(s.loopChecksText()))
	}
}
