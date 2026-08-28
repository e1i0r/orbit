package view

import (
	"reflect"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// foldCase is one ordering of events and the single state it folds into.
// The table is the specification: every ordering this package claims to
// understand is written down here as a want value, not asserted field by
// field, so an ordering that quietly changes one other field fails too.
type foldCase struct {
	name   string
	events []record.Event
	want   Task
}

// at is a fixed timestamp, m minutes into the hour every case starts from.
// The times are literal rather than time.Now() so a want value can be
// written down instead of being computed from whatever the fold did.
func at(m int) time.Time {
	return time.Date(2026, 8, 23, 9, 0, 0, 0, time.UTC).Add(time.Duration(m) * time.Minute)
}

// data is one event's Data map, written as pairs so a case stays on one line.
func data(pairs ...string) map[string]string {
	m := make(map[string]string, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}

	return m
}

// foldCases holds the orderings a run produces when nothing goes wrong.
// The ones where something does are in stopped_test.go, and the ones where
// the record itself is damaged are in damage_test.go.
func foldCases() []foldCase {
	return []foldCase{
		{
			name:   "nothing has been recorded yet",
			events: nil,
			want:   Task{Band: ToDo},
		},
		{
			name: "written down and never started",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx\n\nthe body after the first line is not the title"},
			},
			want: Task{Title: "Retry the webhook on 5xx", Band: ToDo, Since: at(0)},
		},
		{
			name: "one run, start to finish",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task", "worktree", "/w")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "phase.finished", Phase: "implement", Data: data("cost", "0.42")},
				{At: at(4), Kind: "task.finished"},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: Done, Flow: "task",
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(4), Started: at(1), Attempt: 1, Cost: 0.42, state: stateFinished,
			},
		},
		{
			name: "a run inside its first phase",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Reconciliation endpoint"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
			},
			want: Task{
				Title: "Reconciliation endpoint", Band: Running, Flow: "task",
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(2), Started: at(1), Attempt: 1, state: stateRunning,
			},
		},
		{
			name: "a run in flight with live thought and tool call",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Live action task"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "phase.thought", Phase: "implement", Text: "investigating tests"},
				{At: at(4), Kind: "phase.tool_call", Phase: "implement", Data: data("tool", "Bash", "args", "go test ./...")},
			},
			want: Task{
				Title: "Live action task", Band: Running, Flow: "task",
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(2), Started: at(1), Attempt: 1,
				CurrentAction:  "🛠️ Bash: go test ./...",
				CurrentThought: "investigating tests",
				ToolCallCount:  1, state: stateRunning,
			},
		},
		{
			name: "between two phases, the cost of each one summed",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Index on settlements"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "phase.finished", Phase: "implement", Data: data("cost", "0.25")},
				{At: at(4), Kind: "phase.started", Phase: "review", Data: data("engine", "claude", "model", "sonnet", "n", "2")},
				{At: at(5), Kind: "phase.finished", Phase: "review", Data: data("cost", "0.75")},
			},
			want: Task{
				Title: "Index on settlements", Band: Running, Flow: "task",
				Phase: "review", PhaseN: 2, Engine: "claude", Model: "sonnet",
				Since: at(4), Started: at(1), Attempt: 1, Cost: 1.0, state: stateRunning,
			},
		},
		{
			name: "the flow a run overrode the written one with",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Move the assets cron", Data: data("flow", "task")},
				{At: at(1), Kind: "task.started", Data: data("flow", "quick")},
			},
			want: Task{
				Title: "Move the assets cron", Band: Running, Flow: "quick",
				Since: at(1), Started: at(1), Attempt: 1, state: stateRunning,
			},
		},
		{
			name: "finished, and read",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Fix the swagger lint"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "task.finished"},
				{At: at(4), Kind: "task.read"},
			},
			want: Task{
				Title: "Fix the swagger lint", Band: Done, Flow: "task",
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(3), Started: at(1), Attempt: 1, Read: true, state: stateFinished,
			},
		},
		{
			// The instant between run.go:44 and run.go:49, where a run has
			// started and no phase has. A log stays this way if the
			// phase.started emit is what fails (run.go:53-55), and the row
			// must not go on drawing the previous attempt's phase, model and
			// number beside a run that has only just begun.
			name: "a second attempt has no phase until one starts",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "review", Data: data("engine", "claude", "model", "sonnet", "n", "2")},
				{At: at(3), Kind: "phase.failed", Phase: "review", Text: "the test suite is red"},
				{At: at(4), Kind: "task.failed", Text: `task ACME-1, phase "review": the test suite is red`},
				{At: at(5), Kind: "task.started", Data: data("flow", "task")},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: Running, Flow: "task",
				Since: at(5), Started: at(5), Attempt: 2, state: stateRunning,
			},
		},
		{
			name: "a second task.started opens attempt 2 and clears the last one",
			events: []record.Event{
				{At: at(0), Kind: "task.created", Text: "Retry the webhook on 5xx"},
				{At: at(1), Kind: "task.started", Data: data("flow", "task")},
				{At: at(2), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
				{At: at(3), Kind: "phase.failed", Phase: "implement", Text: "the engine said no"},
				{At: at(4), Kind: "task.failed", Text: "task ACME-1, phase implement: the engine said no"},
				{At: at(5), Kind: "task.started", Data: data("flow", "task")},
				{At: at(6), Kind: "phase.started", Phase: "implement", Data: data("engine", "claude", "model", "opus", "n", "1")},
			},
			want: Task{
				Title: "Retry the webhook on 5xx", Band: Running, Flow: "task",
				Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
				Since: at(6), Started: at(5), Attempt: 2, state: stateRunning,
			},
		},
	}
}

func TestFold(t *testing.T) {
	for _, c := range allCases() {
		t.Run(c.name, func(t *testing.T) {
			got := Fold(c.events)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("Fold =\n\t%+v\nwant\n\t%+v", got, c.want)
			}
		})
	}
}

// TestFoldRemembersNothingBetweenCalls folds every case twice, in one order
// and then in the reverse, and requires the same answer both times. The fold
// holding a cache or a package variable is the one defect the table above
// cannot see, because a table run once in a fixed order agrees with a fold
// that remembers.
func TestFoldRemembersNothingBetweenCalls(t *testing.T) {
	cases := allCases()

	first := make([]Task, len(cases))
	for i, c := range cases {
		first[i] = Fold(c.events)
	}

	for i := len(cases) - 1; i >= 0; i-- {
		if got := Fold(cases[i].events); !reflect.DeepEqual(got, first[i]) {
			t.Errorf("%s: folded twice and disagreed:\n\t%+v\nand\n\t%+v", cases[i].name, first[i], got)
		}
	}
}

// allCases is every ordering this package's tests describe, from all three
// tables, so the partition test and the round-trip test above cover the same
// set the fold table does.
func allCases() []foldCase {
	all := foldCases()
	all = append(all, stoppedCases()...)
	all = append(all, stoppedByHandCases()...)
	all = append(all, damageCases()...)

	return all
}
