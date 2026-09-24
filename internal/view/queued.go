package view

// noAttemptYet clears what the current attempt was doing, for a task waiting
// in the queue for a run that has not started. The tool calls and the action
// belong to the attempt, and the overview went on counting the calls of the
// run that had ended.
func noAttemptYet(t *Task) {
	t.CurrentAction, t.CurrentThought, t.ActionKind = "", "", ActionNone
	t.ToolCallCount = 0
}
