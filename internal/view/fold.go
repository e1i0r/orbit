package view

import "github.com/e1i0r/orbit/internal/record"

// whyPaused is the value phase.waiting carries when the reader asked for the
// stop rather than the flow. It is the difference between a task that needs
// you and a task you are holding, and it is the only thing in the record
// that can tell those two apart.
const whyPaused = "paused"

// Fold walks a task's events, oldest first, and returns the one state the
// window draws.
//
// It cannot fail. Every event is either understood, counted as damage, or
// ignored, and there is no input for which the honest answer is an error —
// a log that could not be read at all never reaches here, because there are
// no events to hand over.
//
// Three fields it leaves empty, and one it leaves false:
//
//   - ID, Repo and RepoPath say where the log was found, which is the
//     caller's knowledge and not the log's. internal/board fills them in.
//   - Live says whether a process is believed to hold the task, and the
//     record cannot answer it: a run killed with SIGKILL writes nothing on
//     its way out, so a log whose last event is phase.started is either a
//     run in flight or a run that died. internal/board decides from the pid
//     file, and where the answer is "it died", the fix is to append
//     task.abandoned rather than to display a private conclusion. So the
//     fold reports what the record says — Running — and leaves the promotion
//     to somebody who can look at a process.
func Fold(events []record.Event) Task {
	var t Task
	for _, e := range events {
		fold(&t, e)
	}
	// One conversion, at the end, in one place. The state above is this
	// package's working vocabulary and the band is what leaves it, so Fold
	// is the only thing that maps between them and BandOf reads the result.
	t.Band = bandOfState(t.state)

	return t
}

// fold applies one event. It is separate from the loop so that the loop is
// one line and this is a flat table of kinds, which is the shape a reader
// checks against the writer.
func fold(t *Task, e record.Event) {
	switch e.Kind {
	case record.Unreadable:
		// A line nobody could parse never becomes state. It is counted, and
		// the count is rendered as something the reader can go and look at
		// with their own eyes. Folding a damaged line into a verdict is
		// exactly the lie the whole record exists to prevent: the one thing
		// worse than not knowing what happened is being told confidently.
		t.Damaged++
	case record.TaskCreated:
		// task.created carries the whole of task.md, and the first line of
		// it is the title. Everything below the first line is the task, and
		// the window has a pane for that.
		t.Title = firstLine(e.Text)
		flow(t, e)
		t.state = stateNew
		stamp(&t.Since, e.At)
	case record.TaskStarted:
		// There is no run identifier anywhere in the record; the boundary
		// between one run and the next is this event. Everything the last
		// attempt left on the row goes with it — a phase from an attempt
		// that is over is a stale phase, and a reason that has been retried
		// is not a reason any more. Cost and Read do not: cost is what the
		// task has spent in total, and task.read is a fact about the log.
		//
		// The clearing is what makes a failed re-run honest. internal/task
		// writes this event first, before it validates the flow or makes a
		// worktree, so every attempt reaches here — including the ones
		// refused before a phase — and the phase the attempt before it died
		// in is cleared before the task.failed that refuses this one
		// arrives. It reads as "failed to start", which is what happened.
		//
		// Logs written before that ordering changed have no task.started on
		// the refused attempt at all, and record.TaskFailed below is what
		// catches those.
		t.Attempt++
		t.state = stateRunning
		t.Reason = Reason{}
		t.Phase, t.PhaseN, t.Engine, t.Model = "", 0, "", ""
		t.CurrentAction, t.CurrentThought, t.ActionKind = "", "", ActionNone
		t.ToolCallCount = 0
		flow(t, e)
		stamp(&t.Started, e.At)
		stamp(&t.Since, e.At)
	case record.PhaseStarted:
		t.Phase = e.Phase
		t.PhaseN = count(e.Data["n"])
		// Assigned rather than merged: each phase declares its own engine
		// and model, so an event that does not name one is a record saying
		// there is none, and carrying the last phase's model forward would
		// put a model on a row that never used it.
		t.Engine = e.Data["engine"]
		t.Model = e.Data["model"]
		t.state = stateRunning
		t.Reason = Reason{}
		t.CurrentAction, t.CurrentThought, t.ActionKind = "", "", ActionNone
		stamp(&t.Since, e.At)
	case record.PhaseThought:
		t.CurrentThought = firstLine(e.Text)
		if t.CurrentAction == "" {
			t.CurrentAction, t.ActionKind = firstLine(e.Text), ActionThinking
		}
	case record.PhaseToolCall:
		t.ToolCallCount++
		if act := formatAction(e.Data["tool"], e.Text); act != "" {
			t.CurrentAction, t.ActionKind = act, ActionTool
		}
	case record.PhaseFinished:
		// Since is deliberately not moved. What the row says does not change
		// when a phase ends — the task is still in the same run, and the
		// elapsed number beside it is how long that run has been in this
		// phase. The next phase.started moves it.
		t.Phase = e.Phase
		t.Cost += money(e.Data["cost"])
		t.CurrentAction, t.CurrentThought, t.ActionKind = "", "", ActionNone
	case record.PhaseFailed:
		// Cost is summed from every event that ends a phase, not only the
		// ones that ended well. A phase that ran for twenty minutes and
		// then broke was paid for, and a total that quietly leaves it out
		// is a number the reader will act on and be wrong about.
		// phase.failed is the only place a failure's phase is written down;
		// the task.failed that follows it never carries one. Recording the
		// verdict here as well as there is what makes a log that ends at
		// this event — the next write is best-effort and its error is
		// discarded (run.go:109), so a log can end here — fold to the same
		// band and the same reason as a log that got both.
		//
		// The state is its own, and not stateFailed, because the task.failed
		// that normally follows has to be able to tell "the phase I am
		// holding is this run's" from "the phase I am holding is over". That
		// is the whole difference between the two, and it is not otherwise
		// in the record.
		t.Phase = e.Phase
		t.Cost += money(e.Data["cost"])
		t.state = statePhaseFailed
		t.Reason = failure(e.Phase)
		stamp(&t.Since, e.At)
	case record.TaskFailed:
		// One event, two situations, and internal/task writes the same
		// thing for both from one function. Where the fold already is
		// decides which this is.
		if inAttempt(t.state) {
			// A run was under way, so the phase it is holding is the phase
			// it died in — phase.failed put it there a moment ago.
			t.state = stateFailed
			t.Reason = failure(t.Phase)
		} else {
			// Nothing was under way, so this event landed on the end of a
			// log whose last phase belongs to an attempt that is over.
			// Naming that phase would tell the reader a run died in review
			// when it never started, so the phase goes with the attempt it
			// belonged to.
			//
			// Since internal/task began writing task.started first, a
			// refused re-run reaches here with the phase already cleared and
			// this branch changes nothing. It still matters for two logs:
			// one written before that ordering changed, and one where the
			// run that opened the attempt ended without a task-level event
			// of its own — a phase.failed whose task.failed never got
			// written, and then a re-run.
			t.Phase, t.PhaseN, t.Engine, t.Model = "", 0, "", ""
			t.state = stateFailed
			t.Reason = Reason{Key: ReasonFailedToStart}
		}

		stamp(&t.Since, e.At)
	case record.PhaseCancelled:
		// Cost again: a phase stopped halfway spent whatever it spent, and
		// the reader who stopped it is the one most likely to be asking.
		t.Phase = e.Phase
		t.Cost += money(e.Data["cost"])
		t.state = stateCancelled
		t.Reason = Reason{Key: ReasonCancelled}
		stamp(&t.Since, e.At)
	case record.TaskCancelled:
		t.state = stateCancelled
		t.Reason = Reason{Key: ReasonCancelled}
		stamp(&t.Since, e.At)
	case record.TaskRequeued:
		// Back to where a task sits before anything has run: the attempt is
		// over and there is nothing of it left to draw. What it spent stays
		// — the money and the tokens were spent whoever changed their mind
		// — and so does Attempt, because the next run is the next attempt
		// and not the first one again.
		t.state = stateNew
		t.Reason = Reason{}
		t.Phase, t.PhaseN, t.Engine, t.Model = "", 0, "", ""
		t.CurrentAction, t.CurrentThought, t.ActionKind = "", "", ActionNone
		t.ToolCallCount = 0
		stamp(&t.Since, e.At)
	case record.TaskTimedOut:
		t.state = stateTimedOut
		t.Reason = Reason{Key: ReasonTimedOut}
		stamp(&t.Since, e.At)
	case record.TaskAbandoned:
		t.state = stateAbandoned
		t.Reason = Reason{Key: ReasonAbandoned}
		stamp(&t.Since, e.At)
	case record.PhaseWaiting:
		t.Phase = e.Phase
		t.state, t.Reason = waiting(e)
		stamp(&t.Since, e.At)
	case record.PhaseResumed:
		t.Phase = e.Phase
		t.state = stateRunning
		t.Reason = Reason{}
		stamp(&t.Since, e.At)
	case record.TaskFinished:
		t.state = stateFinished
		t.Reason = Reason{}
		stamp(&t.Since, e.At)
	case record.TaskStuck:
		// The attempts ran out. It is a band of its own way of getting to
		// NeedsYou: a failed run can be retried and this is what is left
		// when retrying is what already happened, so the row says how many
		// were spent rather than which phase broke.
		t.state = stateStuck
		t.Reason = Reason{Key: ReasonStuck, Args: []Arg{{Name: "attempts", Value: e.Data["attempts"]}}}
		stamp(&t.Since, e.At)
	case record.DecisionMade, record.DecisionSuperseded, record.RepoJoined:
		// Written down and deliberately not folded. What was decided and
		// which repositories the task reached are facts about the task, but
		// they are not where the run is: a decision made in the middle of a
		// working run leaves it working. The task view reads these lines
		// from the log itself, which is what keeps them from being folded
		// into a verdict nobody wrote.
	case record.TaskRead:
		// Read says task.read is in the log and nothing more. It does not
		// move Since, because being read does not change what the row says
		// about when this task got where it is.
		t.Read = true
	default:
		// A kind this version has never heard of — including the empty kind
		// a bare `{}` line unmarshals to. It is not damage: the record is
		// append-only and older readers are expected to meet newer writers,
		// and a reader that turned an unfamiliar kind into a verdict would
		// be guessing. Damaged counts lines that are not JSON at all, which
		// is a different fact and one somebody can act on.
	}
}
