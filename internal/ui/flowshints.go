package ui

// One sentence about the field the reader is on.
//
// A form whose labels are the whole explanation is a form somebody has to
// have been told about. These are what a colleague would say while pointing
// at the row: what the field does, and what happens if it is left alone.

// fieldHint is that sentence for the field under the cursor.
func (m Model) fieldHint() string {
	p := m.opts.Words

	switch m.flows.field {
	case flowFieldTemplate:
		return p.T("flows.hint_template", "a starting point: picking one fills the phases below, which you can then change")
	case flowFieldName:
		return p.T("flows.hint_name", "what you will pick this flow by when you start a task")
	case flowFieldDescription:
		return p.T("flows.hint_description", "one line about what this flow is for; it is what the list shows")
	case flowFieldPhaseSelect:
		return p.T("flows.hint_phase_select", "the phases run in this order; the fields below are about the one selected")
	case flowFieldPhaseName:
		return p.T("flows.hint_phase_name", "the name is what the board shows while this phase runs")
	case flowFieldIsLoop:
		return p.T("flows.hint_is_loop", "a repeating phase runs again until its checks pass, and is told what failed each time")
	case flowFieldLoopTurns:
		return p.T("flows.hint_turns", "how many times it may go round before it gives up, so an impossible check cannot run all night")
	case flowFieldLoopUntil:
		return p.T("flows.hint_until", "commands, one per line, as name: command — the loop stops when every one of them exits zero")
	case flowFieldEngine:
		return p.T("flows.hint_engine", "which coding agent runs this phase")
	case flowFieldModel:
		return p.T("flows.hint_model", "the model that engine runs on; left and right walk every one it has")
	case flowFieldEffort:
		return p.T("flows.hint_effort", "how hard the model is asked to think, where the engine offers the choice")
	case flowFieldThinking:
		return p.T("flows.hint_thinking", "adaptive lets Orbit decide from the task; on and off say it outright")
	case flowFieldFeedOutput:
		return p.T("flows.hint_feed", "hand this phase what the phase before it wrote, instead of starting from nothing")
	case flowFieldWait:
		return p.T("flows.hint_wait", "wait stops the flow here for you to look, and nothing runs until you say so")
	case flowFieldPrompt:
		return p.T("flows.hint_prompt", "what this phase is told to do; shift+↵ for a new line, and ✨ writes a first draft")
	case flowFieldAddPhase:
		return p.T("flows.hint_add", "adds a phase after the last one")
	case flowFieldDelPhase:
		return p.T("flows.hint_del", "removes the phase being edited")
	case flowFieldSave:
		return p.T("flows.hint_save", "writes the flow to $ORBIT_HOME/flows and returns to the list")
	}

	return ""
}
