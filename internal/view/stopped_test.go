package view

import "github.com/e1i0r/orbit/internal/record"

// stoppedCases holds every ordering in which a run stops. Four of them are
// the four ways internal/task can fail a run, and three of those write
// task.failed with no Phase on it, because the failure happened before any
// phase started — an invalid flow, an engine nobody configured, a worktree
// that could not be made. They fold to the same shape and are written out
// separately anyway: they are four different sentences a reader will meet,
// and a table that collapses them is a table that stops being the
// specification.
//
// Two more are those same three arriving on a re-run in a log written before
// internal/task began writing task.started first: a bare task.failed on the
// end of a log whose last phase belongs to the attempt before it. They are
// kept because the log is append-only and a reader meets logs older than
// itself. The pair after them is the same re-run in a log written since, and
// the difference between the two pairs is the whole of the fix.
func stoppedCases() []foldCase {
	return []foldCase{
		{
			name: "an invalid flow stops the run before it starts",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.failed", Text: "flow task: phase 2 has no engine"},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: NeedsYou, Since: at(1),
				Reason: Reason{Key: ReasonFailedToStart}, state: stateFailed,
			},
		},
		{
			name: "an engine nobody configured stops it the same way",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.failed", Text: `phase "implement" wants the engine "codex", which is not configured`},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: NeedsYou, Since: at(1),
				Reason: Reason{Key: ReasonFailedToStart}, state: stateFailed,
			},
		},
		{
			name: "a worktree that could not be made stops it the same way",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.failed", Text: "add worktree: branch orbit/ACME-1 is already checked out"},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: NeedsYou, Since: at(1),
				Reason: Reason{Key: ReasonFailedToStart}, state: stateFailed,
			},
		},
		{
			name: "a phase failed, and task.failed carries no phase of its own",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "gates", Data: data("engine", "claude", "model", "opus", "n", "3")},
				{At: at(3), Kind: "phase.failed", Phase: "gates", Text: "the test suite is red"},
				{At: at(4), Kind: "task.failed", Text: "task ACME-1, phase gates: the test suite is red"},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: NeedsYou, Flow: "task",
				Phase: "gates", PhaseN: 3, Engine: "claude", Model: "opus",
				Since: at(4), Started: at(1), Attempt: 1, state: stateFailed,
				Reason: Reason{Key: ReasonFailed, Args: []Arg{{Name: "phase", Value: "gates"}}},
			},
		},
		{
			// A log can end here. The task.failed that normally follows is
			// written best-effort and its error is discarded (run.go:109),
			// so the second write is allowed to be the one that fails.
			name: "a phase failed and nothing was written after it",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "gates", Data: data("engine", "claude", "model", "opus", "n", "3")},
				{At: at(3), Kind: "phase.failed", Phase: "gates", Text: "the test suite is red"},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: NeedsYou, Flow: "task",
				Phase: "gates", PhaseN: 3, Engine: "claude", Model: "opus",
				Since: at(3), Started: at(1), Attempt: 1, state: statePhaseFailed,
				Reason: Reason{Key: ReasonFailed, Args: []Arg{{Name: "phase", Value: "gates"}}},
			},
		},
		{
			// A re-run of a task that finished, refused for an invalid flow,
			// in a log old enough that the refusal wrote no task.started.
			// What reaches the log is a bare task.failed on the end of a
			// finished run. The phase and the model on the row are the
			// finished run's; the run this event is about never had either.
			name: "a re-run whose flow is invalid, after a run that finished",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "phase.finished", Phase: "implement", Data: data("cost", "0.42")},
				{At: at(4), Kind: "task.finished"},
				{At: at(5), Kind: "task.failed", Text: "flow task: phase 2 has no engine"},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: NeedsYou, Flow: "task",
				Since: at(5), Started: at(1), Attempt: 1, Cost: 0.42, state: stateFailed,
				Reason: Reason{Key: ReasonFailedToStart},
			},
		},
		{
			// The same shape after a run that failed: an old log again, with
			// no task.started on the refused attempt. The phase the fold is
			// holding is the one the previous attempt died in, and it is
			// not this run's.
			name: "a re-run whose worktree is busy, after a run that failed",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "review", Data: data("engine", "claude", "model", "sonnet", "n", "2")},
				{At: at(3), Kind: "phase.failed", Phase: "review", Text: "the test suite is red"},
				{At: at(4), Kind: "task.failed", Text: `task ACME-1, phase "review": the test suite is red`},
				{At: at(5), Kind: "task.failed", Text: "add worktree: branch orbit/ACME-1 is already checked out"},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: NeedsYou, Flow: "task",
				Since: at(5), Started: at(1), Attempt: 1, state: stateFailed,
				Reason: Reason{Key: ReasonFailedToStart},
			},
		},
		{
			// The ordering that was a defect, in a log written since the
			// fix. A run was killed outright, so its log ends at
			// phase.started with no terminal event; then a re-run is refused
			// before it reaches a phase. task.started now lands first, and
			// it takes the dead attempt's phase with it — where before, the
			// fold was still inside that attempt and read the refusal as a
			// failure in a phase this run never entered.
			name: "a re-run refused before any phase, on a log a kill left open",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "review", Data: data("engine", "claude", "model", "sonnet", "n", "2")},
				{At: at(3), Kind: "task.started", Data: data("flow", "task")},
				{At: at(4), Kind: "task.failed", Text: "flow task: phase 2 has no engine"},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: NeedsYou, Flow: "task",
				Since: at(4), Started: at(3), Attempt: 2, state: stateFailed,
				Reason: Reason{Key: ReasonFailedToStart},
			},
		},
		{
			// The same fix over the other log that can end mid-attempt: a
			// phase.failed whose task.failed was never written, because that
			// write is best-effort and its error is discarded.
			name: "a re-run refused before any phase, after a phase.failed nothing followed",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "review", Data: data("engine", "claude", "model", "sonnet", "n", "2")},
				{At: at(3), Kind: "phase.failed", Phase: "review", Text: "the test suite is red", Data: data("cost", "0.30")},
				{At: at(4), Kind: "task.started", Data: data("flow", "task")},
				{At: at(5), Kind: "task.failed", Text: "add worktree: branch orbit/ACME-1 is already checked out"},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: NeedsYou, Flow: "task",
				Since: at(5), Started: at(4), Attempt: 2, Cost: 0.30, state: stateFailed,
				Reason: Reason{Key: ReasonFailedToStart},
			},
		},
		{
			// A phase that broke was still paid for, and so was a phase
			// somebody stopped. A total that counted only the phases that
			// ended well would be a number the reader acts on and is wrong
			// about.
			name: "the cost of a phase that failed is still the cost",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Index on settlements"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "phase.finished", Phase: "implement", Data: data("cost", "0.25")},
				{At: at(4), Kind: "phase.started", Phase: "review", Data: data("engine", "claude", "model", "sonnet", "n", "2")},
				{At: at(5), Kind: "phase.failed", Phase: "review", Text: "the test suite is red", Data: data("cost", "0.75", "error", "exit status 1")},
				{At: at(6), Kind: "task.failed", Text: `task ACME-1, phase "review": exit status 1`},
			},
			want: Task{
				Title: "Index on settlements", Band: NeedsYou, Flow: "task",
				Phase: "review", PhaseN: 2, Engine: "claude", Model: "sonnet",
				Since: at(6), Started: at(1), Attempt: 1, Cost: 1.0, state: stateFailed,
				Reason: Reason{Key: ReasonFailed, Args: []Arg{{Name: "phase", Value: "review"}}},
			},
		},
		{
			name: "the cost of a phase that was stopped is still the cost",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Index on settlements"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "phase.cancelled", Phase: "implement", Text: "half of it", Data: data("cost", "0.40")},
				{At: at(4), Kind: "task.cancelled"},
			},
			want: Task{
				Title: "Index on settlements", Band: Done, Flow: "task",
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(4), Started: at(1), Attempt: 1, Cost: 0.40, state: stateCancelled,
				Reason: Reason{Key: ReasonCancelled},
			},
		},
	}
}
