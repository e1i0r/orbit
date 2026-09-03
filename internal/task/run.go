package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Run walks a flow, phase by phase, in a worktree of its own.
//
// It stops at the first failure and says why, in the words the engine used,
// and it stops before any phase its gate will not let past. A nil gate never
// stops anything, which is what `orbit run` was before there was a window to
// release a run from.
//
// The worktree is never removed. Not on failure, where the work that did
// happen is the most valuable thing in the run and is not this function's to
// throw away — and not on success either, where the design's answer to a
// phase that settles green is to let a human take the keyboard and carry on
// in that same checkout. This is worth saying plainly because the doc
// comment here once said "never removed on failure", which implied a
// cleanup on success that does not exist: Orbit has no verb that removes a
// settled worktree. repo.RemoveWorktree is written and nothing calls it, so
// every run leaves a .git/worktrees entry behind in a repository Orbit does
// not own, and `git worktree prune` by hand is the only remedy.
func Run(ctx context.Context, s *store.Store, t Task, f flow.Flow, engines map[string]engine.Engine, g Gate) error {
	// The run marker goes down before anything is written, and comes off on
	// every way out of here. It is what lets any reader tell a phase still
	// running from a phase whose process is gone, and it is also the only
	// thing that can say a task is already taken — so a second run of one
	// task is refused here, before this attempt has put a single line in a
	// log that belongs to the run already walking it. hold says why.
	//
	// Nothing is recorded when it refuses, and that is the point. The log
	// already describes the run that is happening; an attempt that never
	// began has nothing to add to it, and a task.started followed by a
	// task.failed would tell every reader that the *other* run had ended.
	//
	// The marker cannot survive SIGKILL — nothing written by the dying
	// process can — so the invariant a reader may rely on is the weaker,
	// true one: a task's log ends in a terminal event, or a reader appends
	// one. Reconcile is the reader that appends it.
	release, err := hold(s, t)
	if err != nil {
		return noted(t.ID, err)
	}
	defer release()

	// task.started is written next — before the flow is validated, before
	// the engines are checked, before the worktree exists — and the ordering
	// is load-bearing rather than tidy.
	//
	// Every way a run can fail writes task.failed (see failed below), and a
	// task.failed lands on the end of a log that may already carry a phase
	// from the attempt before it. The event that tells a reader the old
	// phase is over is task.started, which clears it. If the three
	// pre-phase failures returned before task.started were ever written, a
	// second attempt refused for an invalid flow would show up in the window
	// as a failure in the phase the *first* attempt died in: a stale phase,
	// named after an attempt that never ran. Written here it needs no reader
	// to change, and this is where an attempt begins — so the log has a line
	// for every attempt rather than only for the ones that reached a
	// worktree.
	//
	// It costs the worktree path this event could otherwise carry. Nothing
	// reads it, and it is a pure function of the state root, the
	// repository's path and the id (store.CreateWorktreeParent), so nothing
	// would be knowable only from that field — whereas an attempt with no
	// line in the log is knowable from nowhere at all.
	if err := emit(s, t, record.Event{Kind: record.TaskStarted, Data: map[string]string{"flow": f.Name}}); err != nil {
		return err
	}

	if err := runnable(f, engines); err != nil {
		return failed(s, t, err)
	}

	wt, err := prepare(s, t)
	if err != nil {
		return failed(s, t, err)
	}

	// The listing is read once and given to every phase, rather than once
	// per phase: a repository cloned while a run is walking its flow is not
	// something this run was asked about, and a phase whose prompt differs
	// from the one before it for a reason nobody caused is a run that cannot
	// be read back. A workspace that cannot be walked is not a failure —
	// the task is still the task, and the phase runs with no listing.
	//
	// It is read after prepare and not before, so that a task with no
	// repository has its directory before it is told what it could join.
	others := elsewhere(s, t)

	var prevOutput string

	for i, p := range f.Phases {
		// Every phase is put to the gate, not only the ones whose Wait says
		// so; Gate says why. A gate that cannot read what it needs is a
		// failure of the run, because a run that cannot be held is not the
		// run the reader asked for.
		decision, gateErr := ask(ctx, g, t, p, i+1)
		if gateErr != nil {
			return failed(s, t, fmt.Errorf("task %s, before phase %q: %w", t.ID, p.Name, gateErr))
		}

		switch decision {
		case Stop:
			// A context that is done is not a reader who said cancel, and
			// asking the context is how they are told apart — the same
			// distinction the engine's error path draws below.
			return gateStop(s, t, p.Name, ctx.Err())
		case Skip:
			// Nothing is recorded for a phase that did not run. A phase.
			// started with no ending would read for ever as a phase in
			// flight, and a phase.finished would be a lie about work.
			continue
		}

		// The budget is asked before the notes are taken, so that a run
		// stopped by it does not consume a note the next run will need.
		if spent, budget, over := overBudget(s, t); over {
			return stopSpending(s, t, p, spent, budget)
		}

		notes, notesErr := unconsumedNotes(s, t)
		if notesErr != nil {
			return failed(s, t, fmt.Errorf("task %s, before phase %q: %w", t.ID, p.Name, notesErr))
		}

		out, err := attempts(ctx, phaseRun{
			store:  s,
			task:   t,
			phase:  p,
			eng:    engines[p.Engine],
			n:      i + 1,
			wt:     wt,
			notes:  notes,
			prev:   fedOutput(p, prevOutput),
			others: others,
		}, f.AttemptCap())
		if err != nil {
			return err
		}

		// Only when there is something to carry. A phase that finished
		// silently leaves the last real answer standing rather than blanking
		// it, so the "Previous Phase Output" the next prompt shows can be two
		// phases back. That is the intended trade and it is worth stating,
		// because the label does not: the alternative hands a phase that asked
		// to be fed nothing at all, and a stale answer is worth more to it
		// than an empty one. Which phase actually said it is in the log, where
		// every phase's output is a line of its own.
		if out.Output != "" {
			prevOutput = out.Output
		}
	}

	// Every phase ran and every one of them was written down; only the line
	// that says so did not land. Calling that a failure is the closest true
	// thing the record can say — the run did not complete, because
	// completing includes being able to report it — and it is far better
	// than the alternative, which is a finished run whose log stays open for
	// ever because its last write was the one that failed.
	if err := emit(s, t, record.Event{Kind: record.TaskFinished}); err != nil {
		return failed(s, t, err)
	}

	return nil
}

// stopped writes down that a run was stopped from outside.
func stopped(s *store.Store, t Task, phase string, out engine.Result, cause error) error {
	_ = emit(s, t, phaseEnd(record.PhaseCancelled, phase, out, nil)) //nolint:errcheck

	kind := record.TaskCancelled
	if errors.Is(cause, context.DeadlineExceeded) {
		kind = record.TaskTimedOut
	}

	_ = emit(s, t, record.Event{Kind: kind}) //nolint:errcheck

	return fmt.Errorf("task %s, phase %q: %w", t.ID, phase, cause)
}

// failed writes down that the run stopped and why.
func failed(s *store.Store, t Task, err error) error {
	text, _ := captured(err.Error())
	_ = emit(s, t, record.Event{Kind: record.TaskFailed, Text: text}) //nolint:errcheck

	return err
}
