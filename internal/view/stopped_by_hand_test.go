package view

import "github.com/e1i0r/orbit/internal/record"

// stoppedCases in stopped_test.go holds the orderings where a run broke.
// These are the orderings where something stopped it, or held it, or is
// waiting on somebody: a reader who cancelled it, a deadline that passed, a
// gate, and the record of a run whose process was gone and a reader closed
// it. The split is where the file ran past what one file may weigh; the two
// tables are read as one specification and are run as one by TestFold.
func stoppedByHandCases() []foldCase {
	return []foldCase{
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
			// The one gesture that moves a task backwards. What it spent is
			// still in the record and the attempt still counts, so the next
			// run is the second one and not the first over again.
			name: "the reader took it back, and a requeued task is to do again",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Move the assets cron"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "phase.cancelled", Phase: "implement", Data: data("cost", "0.42")},
				{At: at(4), Kind: "task.cancelled"},
				{At: at(5), Kind: "task.requeued", Text: "wrong engine", Data: data("by", "operator")},
			},
			want: Task{
				Title: "Move the assets cron", Band: ToDo, Flow: "task",
				Since: at(5), Started: at(1), Attempt: 1, Cost: 0.42, state: stateNew,
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
