package view

import "github.com/e1i0r/orbit/internal/record"

// stoppedCases holds every ordering in which a run stops. Four of them are
// the four ways internal/task can fail a run, and three of those write
// task.failed with no Phase on it, because the failure happened before any
// phase started — an invalid flow, an engine nobody configured, a worktree
// that could not be made. They fold to the same shape and are written out
// separately anyway: they are four different sentences a reader will meet,
// and a table that collapses them is a table that stops being the
// specification. The fifth is those same three arriving on a second attempt,
// where the phase of the attempt before is still in living memory.
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
			// The ordering the second attempt makes: a run that failed inside
			// a phase, retried, and then failed again before the new attempt
			// reached one. The phase the *previous* attempt died in must not
			// be the phase this row names, or the screen tells the reader a
			// run died in review when it never got past starting. It is not a
			// hypothetical ordering: three of the four ways internal/task
			// fails a run write exactly it.
			name: "a re-run that fails before a phase starts names no phase",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "review", Data: data("engine", "claude", "model", "opus", "n", "2")},
				{At: at(3), Kind: "phase.failed", Phase: "review", Text: "the test suite is red"},
				{At: at(4), Kind: "task.failed", Text: "task ACME-1, phase review: the test suite is red"},
				{At: at(5), Kind: "task.started", Data: data("flow", "quick")},
				{At: at(6), Kind: "task.failed", Text: `phase "implement" wants the engine "codex", which is not configured`},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: NeedsYou, Flow: "quick",
				Since: at(6), Started: at(5), Attempt: 2, state: stateFailed,
				Reason: Reason{Key: ReasonFailedToStart},
			},
		},
		{
			name: "the reader cancelled it, and a cancelled task is done",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Move the assets cron"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "phase.cancelled", Phase: "implement", Text: "what it had printed before it was stopped"},
				{At: at(4), Kind: "task.cancelled"},
			},
			want: Task{
				Title: "Move the assets cron", Band: Done, Flow: "task",
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(4), Started: at(1), Attempt: 1, state: stateCancelled,
				Reason: Reason{Key: ReasonCancelled},
			},
		},
		{
			name: "it ran past its timeout, and a timeout needs you",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Move the assets cron"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "task.timedout"},
			},
			want: Task{
				Title: "Move the assets cron", Band: NeedsYou, Flow: "task",
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(3), Started: at(1), Attempt: 1, state: stateTimedOut,
				Reason: Reason{Key: ReasonTimedOut},
			},
		},
		{
			name: "the flow asked the phase to wait, which needs you",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Move the assets cron"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "review", Data: data("engine", "claude", "model", "opus", "n", "2")},
				{At: at(3), Kind: "phase.waiting", Phase: "review", Data: data("why", "flow")},
			},
			want: Task{
				Title: "Move the assets cron", Band: NeedsYou, Flow: "task",
				Phase: "review", PhaseN: 2, Engine: "claude", Model: "opus",
				Since: at(3), Started: at(1), Attempt: 1, state: stateWaiting,
				Reason: Reason{Key: ReasonGate, Args: []Arg{{Name: "phase", Value: "review"}}},
			},
		},
		{
			name: "the reader asked it to hold, which is still running",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Move the assets cron"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "review", Data: data("engine", "claude", "model", "opus", "n", "2")},
				{At: at(3), Kind: "phase.waiting", Phase: "review", Data: data("why", "paused")},
			},
			want: Task{
				Title: "Move the assets cron", Band: Running, Flow: "task",
				Phase: "review", PhaseN: 2, Engine: "claude", Model: "opus",
				Since: at(3), Started: at(1), Attempt: 1, state: stateHeld,
				Reason: Reason{Key: ReasonHeld, Args: []Arg{{Name: "phase", Value: "review"}}},
			},
		},
		{
			name: "resumed, and the reason it was held goes with it",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Move the assets cron"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "review", Data: data("engine", "claude", "model", "opus", "n", "2")},
				{At: at(3), Kind: "phase.waiting", Phase: "review", Data: data("why", "paused")},
				{At: at(4), Kind: "phase.resumed", Phase: "review", Data: data("word", "resume")},
			},
			want: Task{
				Title: "Move the assets cron", Band: Running, Flow: "task",
				Phase: "review", PhaseN: 2, Engine: "claude", Model: "opus",
				Since: at(4), Started: at(1), Attempt: 1, state: stateRunning,
			},
		},
		{
			name: "its process is gone and a reader wrote that down",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Fix the swagger lint"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "task.abandoned"},
			},
			want: Task{
				Title: "Fix the swagger lint", Band: NeedsYou, Flow: "task",
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(3), Started: at(1), Attempt: 1, state: stateAbandoned,
				Reason: Reason{Key: ReasonAbandoned},
			},
		},
	}
}
