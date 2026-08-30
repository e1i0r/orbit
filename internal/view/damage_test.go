package view

import "github.com/e1i0r/orbit/internal/record"

// damageCases holds the logs that are not clean: a line nobody could parse,
// an event with no kind, a kind this version has never heard of, a number
// that is not one, and a timestamp that is the zero time. None of them may
// become a verdict, and the want values below are how that is held to.
func damageCases() []foldCase {
	return []foldCase{
		{
			name: "a line nobody could parse is counted and nothing else",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{Kind: record.Unreadable, Text: "this line of the record is not valid JSON and was skipped", Data: data("line", "2")},
				{At: at(2), Kind: "task.started", Data: data("flow", "task")},
				{At: at(3), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: Running, Flow: "task",
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(3), Started: at(2), Attempt: 1, Damaged: 1, state: stateRunning,
			},
		},
		{
			name: "a damaged line after a finish does not un-finish it",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Fix the swagger lint"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "task.finished"},
				{Kind: record.Unreadable, Data: data("line", "4")},
				{Kind: record.Unreadable, Data: data("line", "5")},
			},
			want: Task{
				Title: "Fix the swagger lint", Band: Done, Flow: "task",
				Since: at(2), Started: at(1), Attempt: 1, Damaged: 2, state: stateFinished,
			},
		},
		{
			// A task written with CR line endings, which is how one arrives
			// from an editor that has never seen a newline. The title is one
			// row of the board beside four columns, and everything after the
			// return would be drawn on top of them.
			name: "a title whose lines end in carriage returns is only its first line",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Add orbit version to the CLI\r- write internal/cli/version.go\r- write its tests"},
			},
			want: Task{Title: "Add orbit version to the CLI", Band: ToDo, Since: at(0)},
		},
		{
			name: "a title whose lines end in both is only its first line",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Add orbit version to the CLI\r\n- write internal/cli/version.go"},
			},
			want: Task{Title: "Add orbit version to the CLI", Band: ToDo, Since: at(0)},
		},
		{
			name: "a line that is only {} unmarshals to an event with no kind",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Index on settlements"},
				{},
			},
			want: Task{Title: "Index on settlements", Band: ToDo, Since: at(0)},
		},
		{
			name: "a kind this version has never heard of is not damage",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Index on settlements"},
				{At: at(1), Kind: "task.updated", Text: "written by a version that came after this one"},
			},
			want: Task{Title: "Index on settlements", Band: ToDo, Since: at(0)},
		},
		{
			name: "a phase.finished with no Data at all",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Index on settlements"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "phase.finished", Phase: "implement"},
			},
			want: Task{
				Title: "Index on settlements", Band: Running, Flow: "task",
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(2), Started: at(1), Attempt: 1, state: stateRunning,
			},
		},
		{
			name: "a cost and a phase number that are not numbers",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Index on settlements"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "n", "first")},
				{At: at(3), Kind: "phase.finished", Phase: "implement", Data: data("cost", "free")},
				{At: at(4), Kind: "phase.finished", Phase: "implement", Data: data("cost", "NaN")},
				{At: at(5), Kind: "phase.finished", Phase: "implement", Data: data("cost", "-3")},
			},
			want: Task{
				Title: "Index on settlements", Band: Running, Flow: "task",
				Phase: "implement", Engine: "claude",
				Since: at(2), Started: at(1), Attempt: 1, state: stateRunning,
			},
		},
		{
			name: "a zero timestamp never becomes a time on screen",
			events: []record.Event{
				{Kind: "task.created", Text: "Move the assets cron"},
				{Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{Kind: "task.finished"},
			},
			want: Task{
				Title: "Move the assets cron", Band: Done, Flow: "task",
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(2), Attempt: 1, state: stateFinished,
			},
		},
		{
			name: "a phase.waiting that does not say why needs you",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Move the assets cron"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "review", Data: data("engine", "claude", "model", "opus", "n", "2")},
				{At: at(3), Kind: "phase.waiting", Phase: "review"},
			},
			want: Task{
				Title: "Move the assets cron", Band: NeedsYou, Flow: "task",
				Phase: "review", PhaseN: 2, Engine: "claude", Model: "opus",
				Since: at(3), Started: at(1), Attempt: 1, state: stateWaiting,
				Reason: Reason{Key: ReasonGate, Args: []Arg{{Name: "phase", Value: "review"}}},
			},
		},
		{
			name: "a phase with no run in front of it is folded as it reads",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Move the assets cron"},
				{At: at(1), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
			},
			want: Task{
				Title: "Move the assets cron", Band: Running,
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(1), state: stateRunning,
			},
		},
	}
}
