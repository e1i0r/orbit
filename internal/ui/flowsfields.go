package ui

// Which fields the designer shows, and what typing into one does.
//
// The list is asked for rather than counted, because it is no longer fixed:
// a phase that repeats shows two fields a phase that does not has nothing to
// say about. Tab, the mouse and the draw all read this one list, so a field
// that appears cannot appear in one of them and not the others.

import "slices"

// fieldsShown is every field of the form, in the order it is drawn.
func (st *flowsState) fieldsShown() []int {
	out := []int{
		flowFieldTemplate,
		flowFieldName,
		flowFieldDescription,
		flowFieldPhaseSelect,
		flowFieldPhaseName,
		flowFieldIsLoop,
	}

	if st.looping() {
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
func (st *flowsState) moveField(delta int) {
	fields := st.fieldsShown()

	at := slices.Index(fields, st.field)
	if at < 0 {
		st.field = flowFieldIsLoop
		return
	}

	st.field = fields[(at+delta+len(fields))%len(fields)]
}

// typing is whether the field under the cursor takes characters, as opposed
// to a dial that left and right turn.
func (st *flowsState) typing() bool {
	switch st.field {
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
func (st *flowsState) multiline() bool {
	switch st.field {
	case flowFieldDescription, flowFieldPrompt, flowFieldLoopUntil:
		return true
	}

	return false
}

// write puts what was typed at the end of the field under the cursor.
func (st *flowsState) write(text string) {
	switch st.field {
	case flowFieldName:
		st.flowName += text
	case flowFieldDescription:
		st.description += text
	case flowFieldPhaseName:
		st.cur().Name += text
	case flowFieldPrompt:
		st.edited().Prompt += text
	case flowFieldLoopUntil:
		st.setChecks(st.loopChecksText() + text)
	}
}

// rub takes the last character back out of it.
func (st *flowsState) rub() {
	switch st.field {
	case flowFieldName:
		st.flowName = trimLastRune(st.flowName)
	case flowFieldDescription:
		st.description = trimLastRune(st.description)
	case flowFieldPhaseName:
		st.cur().Name = trimLastRune(st.cur().Name)
	case flowFieldPrompt:
		st.edited().Prompt = trimLastRune(st.edited().Prompt)
	case flowFieldLoopUntil:
		st.setChecks(trimLastRune(st.loopChecksText()))
	}
}
