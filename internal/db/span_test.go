package db

// The two spans an event sits inside, and the rule that makes them safe to
// store: each of them begins and ends on exactly one event, written by the
// transaction that inserts it.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// history appends a list of events to one task, stamped in order.
func history(t *testing.T, d *DB, taskID string, events ...record.Event) {
	t.Helper()

	tick := clock()

	for _, e := range events {
		e.At = tick()
		if err := d.Append(taskID, e); err != nil {
			t.Fatalf("append %s: %v", e.Kind, err)
		}
	}
}

// phase is a phase.started carrying the two facts a reader wants from it.
func phase(name, engine, model string) record.Event {
	return record.Event{
		Kind:  record.PhaseStarted,
		Phase: name,
		Data:  map[string]string{"engine": engine, "model": model},
	}
}

// TestARunIsOpenedAndClosedByTheEventsThatSaySo.
func TestARunIsOpenedAndClosedByTheEventsThatSaySo(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"},
		record.Event{Kind: record.TaskStarted},
		record.Event{Kind: record.TaskFinished},
	)

	runs, err := d.Runs("ACME-1")
	if err != nil {
		t.Fatalf("read the runs: %v", err)
	}

	if len(runs) != 1 {
		t.Fatalf("the task has %d runs, want one", len(runs))
	}

	r := runs[0]
	if r.N != 1 || r.Outcome != record.TaskFinished {
		t.Errorf("run %d ended %q, want run 1 ending task.finished", r.N, r.Outcome)
	}

	if !r.Ended.After(r.Started) {
		t.Errorf("the run ran from %v to %v, want an end after its start", r.Started, r.Ended)
	}
}

// TestAnEventBeforeTheFirstRunBelongsToNoRun. task.created is written when
// somebody writes a task down, which can be days before anything runs.
func TestAnEventBeforeTheFirstRunBelongsToNoRun(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "Later"})

	var runs int
	if err := d.sql.QueryRow(`SELECT count(*) FROM run`).Scan(&runs); err != nil {
		t.Fatalf("count the runs: %v", err)
	}

	if runs != 0 {
		t.Errorf("a task written down and not started has %d runs, want none", runs)
	}

	var open int
	if err := d.sql.QueryRow(`SELECT count(*) FROM event WHERE run_id IS NULL`).Scan(&open); err != nil {
		t.Fatalf("count the events outside a run: %v", err)
	}

	if open != 1 {
		t.Errorf("%d events sit outside a run, want the one written before any", open)
	}
}

// TestARetryIsASecondRunAndNotTheFirstReopened. task.started is the boundary
// between one attempt and the next — that is the whole reason a run is a row
// rather than a flag on the task.
func TestARetryIsASecondRunAndNotTheFirstReopened(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskStarted},
		record.Event{Kind: record.TaskFailed, Text: "the gate did not pass"},
		record.Event{Kind: record.TaskStarted},
		record.Event{Kind: record.TaskFinished},
	)

	runs, err := d.Runs("ACME-1")
	if err != nil {
		t.Fatalf("read the runs: %v", err)
	}

	if len(runs) != 2 {
		t.Fatalf("the task has %d runs, want two", len(runs))
	}

	if runs[0].N != 1 || runs[0].Outcome != record.TaskFailed {
		t.Errorf("the first attempt is run %d ending %q, want run 1 ending task.failed", runs[0].N, runs[0].Outcome)
	}

	if runs[1].N != 2 || runs[1].Outcome != record.TaskFinished {
		t.Errorf("the second attempt is run %d ending %q, want run 2 ending task.finished", runs[1].N, runs[1].Outcome)
	}
}

// TestAPhaseCarriesWhatRanIt. Which engine and which model is the question
// the cost of a task is answered from, and it is on the phase because one
// task can be worked by two of them.
func TestAPhaseCarriesWhatRanIt(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskStarted},
		phase("plan", "claude", "claude-opus-5"),
		record.Event{Kind: record.PhaseFinished, Phase: "plan"},
		phase("implement", "codex", "gpt-5"),
		record.Event{Kind: record.PhaseFailed, Phase: "implement"},
		record.Event{Kind: record.TaskFailed},
	)

	runs, err := d.Runs("ACME-1")
	if err != nil {
		t.Fatalf("read the runs: %v", err)
	}

	phases := runs[0].Phases
	if len(phases) != 2 {
		t.Fatalf("the run has %d phases, want two", len(phases))
	}

	if phases[0].Name != "plan" || phases[0].Engine != "claude" || phases[0].Model != "claude-opus-5" {
		t.Errorf("the first phase is %+v, want plan on claude", phases[0])
	}

	if phases[0].N != 1 || phases[1].N != 2 {
		t.Errorf("the phases are numbered %d and %d, want 1 and 2", phases[0].N, phases[1].N)
	}

	if phases[1].Outcome != record.PhaseFailed {
		t.Errorf("the second phase ended %q, want phase.failed", phases[1].Outcome)
	}
}

// TestWhatHappensInsideAPhaseIsFiledUnderIt. Every line an engine printed
// arrives between a phase.started and its ending, and the point of the row
// is that those lines can be asked for by phase without folding the task.
func TestWhatHappensInsideAPhaseIsFiledUnderIt(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskStarted},
		phase("implement", "claude", "claude-opus-5"),
		record.Event{Kind: record.PhaseToolCall, Text: "Edit"},
		record.Event{Kind: record.PhaseThought, Text: "the retry belongs in the client"},
		record.Event{Kind: record.PhaseFinished},
	)

	// The phase.started, the two lines inside it, and the phase.finished:
	// the opening and closing events belong to the phase they bound.
	var inside int
	if err := d.sql.QueryRow(`SELECT count(*) FROM event WHERE phase_id IS NOT NULL`).Scan(&inside); err != nil {
		t.Fatalf("count the events inside a phase: %v", err)
	}

	if inside != 4 {
		t.Errorf("%d events are filed under a phase, want the four between and including its ends", inside)
	}

	// And the task.started before it is not one of them.
	var outside int
	if err := d.sql.QueryRow(
		`SELECT count(*) FROM event WHERE phase_id IS NULL AND run_id IS NOT NULL`,
	).Scan(&outside); err != nil {
		t.Fatalf("count the events in the run and outside a phase: %v", err)
	}

	if outside != 1 {
		t.Errorf("%d events are in the run and outside a phase, want the task.started", outside)
	}
}

// TestARunThatEndsLeavesNoPhaseRunning. A phase whose own ending never
// arrived is a process that was killed, and leaving it open would file every
// later event of the task under a phase that stopped.
func TestARunThatEndsLeavesNoPhaseRunning(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskStarted},
		phase("implement", "claude", "claude-opus-5"),
		record.Event{Kind: record.TaskAbandoned, Text: "its process is gone"},
	)

	runs, err := d.Runs("ACME-1")
	if err != nil {
		t.Fatalf("read the runs: %v", err)
	}

	p := runs[0].Phases[0]
	if p.Ended.IsZero() || p.Outcome != record.TaskAbandoned {
		t.Errorf("the phase ended %v as %q, want it closed by the event that ended the run", p.Ended, p.Outcome)
	}

	// The next attempt opens its own phases, numbered from one again.
	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskStarted},
		phase("implement", "claude", "claude-opus-5"),
	)

	runs, err = d.Runs("ACME-1")
	if err != nil {
		t.Fatalf("read the runs: %v", err)
	}

	if n := runs[1].Phases[0].N; n != 1 {
		t.Errorf("the second attempt's first phase is numbered %d, want 1", n)
	}
}

// TestARunningRunAndPhaseHaveNotEnded. The zero time is how a reader tells
// what is happening now from what happened, and it is the only thing the
// band of a task is folded from — there is no column claiming it.
func TestARunningRunAndPhaseHaveNotEnded(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskStarted},
		phase("implement", "claude", "claude-opus-5"),
		record.Event{Kind: record.PhaseToolCall, Text: "Bash"},
	)

	runs, err := d.Runs("ACME-1")
	if err != nil {
		t.Fatalf("read the runs: %v", err)
	}

	if !runs[0].Ended.IsZero() || runs[0].Outcome != "" {
		t.Errorf("a running attempt reads as ended %v, %q", runs[0].Ended, runs[0].Outcome)
	}

	if !runs[0].Phases[0].Ended.IsZero() {
		t.Errorf("a running phase reads as ended %v", runs[0].Phases[0].Ended)
	}
}

// TestAPhaseEndingWithNoneOpenChangesNothing. A phase.finished with nothing
// before it is a malformed history — a log cut at the front, or a migration
// half done — and it is written down like any other event rather than
// closing a phase that is not there.
func TestAPhaseEndingWithNoneOpenChangesNothing(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskStarted},
		record.Event{Kind: record.PhaseFinished, Phase: "implement"},
	)

	var phases int
	if err := d.sql.QueryRow(`SELECT count(*) FROM phase`).Scan(&phases); err != nil {
		t.Fatalf("count the phases: %v", err)
	}

	if phases != 0 {
		t.Errorf("an ending with no phase open made %d phases, want none", phases)
	}

	events, err := d.Events("ACME-1")
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("%d events were written, want both of them", len(events))
	}
}

// TestARefusedAttemptEndsItsPhase is the span half of the attempt cap: a
// phase whose gate refused it is run again, and the row for the attempt that
// was refused has to end where it ended. Left open, it would be closed by
// the run's own outcome — so a task that went on to finish would carry two
// attempts reading "task.finished", stamped at a time neither of them ran.
func TestARefusedAttemptEndsItsPhase(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"},
		record.Event{Kind: record.TaskStarted},
		phase("implement", "claude", "sonnet"),
		record.Event{Kind: record.GateFailed, Phase: "implement"},
		record.Event{Kind: record.PhaseRetried, Phase: "implement"},
		phase("implement", "claude", "sonnet"),
		record.Event{Kind: record.PhaseFinished, Phase: "implement"},
		record.Event{Kind: record.TaskFinished},
	)

	rows, err := d.sql.Query(`SELECT n, outcome, ended_at IS NOT NULL FROM phase ORDER BY n`)
	if err != nil {
		t.Fatalf("read the phases: %v", err)
	}
	defer rows.Close()

	var got []string

	for rows.Next() {
		var (
			n       int
			outcome string
			ended   bool
		)

		if err := rows.Scan(&n, &outcome, &ended); err != nil {
			t.Fatalf("scan a phase: %v", err)
		}

		if !ended {
			t.Errorf("phase %d is still open", n)
		}

		got = append(got, outcome)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("walk the phases: %v", err)
	}

	want := []string{record.PhaseRetried, record.PhaseFinished}
	if len(got) != len(want) {
		t.Fatalf("the run holds %d phases (%v), want two: the attempt that was refused and the one that stood", len(got), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("phase %d ended %q, want %q", i+1, got[i], want[i])
		}
	}
}
