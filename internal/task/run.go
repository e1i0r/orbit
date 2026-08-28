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
		return err
	}
	defer release()

	// task.started is written next — before the flow is validated, before
	// the engines are checked, before the worktree exists — and the ordering
	// is load-bearing rather than tidy.
	//
	// Every way a run can fail writes task.failed (see failed below), and a
	// task.failed lands on the end of a log that may already carry a phase
	// from the attempt before it. The event that tells a reader the old
	// phase is over is task.started, which clears it. While the three
	// pre-phase failures returned before task.started was ever written, a
	// second attempt refused for an invalid flow showed up in the window as
	// a failure in the phase the *first* attempt had died in: a stale phase,
	// named after an attempt that never ran. Writing task.started here is
	// the whole fix, it needs no reader to change, and it is simply true —
	// this is where an attempt begins, and now the log has a line for every
	// attempt rather than only for the ones that reached a worktree.
	//
	// It costs the worktree path this event used to carry. Nothing read it,
	// and it is a pure function of the state root, the repository's path and
	// the id (store.CreateWorktreeParent), so nothing was knowable only from
	// that field — whereas an attempt with no line in the log is knowable
	// from nowhere at all.
	if err := emit(s, t, record.Event{Kind: record.TaskStarted, Data: map[string]string{"flow": f.Name}}); err != nil {
		return err
	}

	if err := f.Validate(); err != nil {
		return failed(s, t, err)
	}

	for _, p := range f.Phases {
		eng, ok := engines[p.Engine]
		if !ok {
			return failed(s, t, fmt.Errorf("phase %q wants the engine %q, which is not configured", p.Name, p.Engine))
		}

		if p.Model != "" && len(eng.Models()) > 0 {
			var found bool

			for _, m := range eng.Models() {
				if m.ID == p.Model {
					found = true
					break
				}
			}

			if !found {
				return failed(s, t, fmt.Errorf("phase %q names model %q, which engine %q does not offer", p.Name, p.Model, p.Engine))
			}
		}

		if p.Effort != "" {
			efforts := eng.Efforts()
			if len(efforts) == 0 {
				return failed(s, t, fmt.Errorf("phase %q names effort %q, but engine %q has no effort dial", p.Name, p.Effort, p.Engine))
			}

			var found bool

			for _, e := range efforts {
				if e.ID == p.Effort {
					found = true
					break
				}
			}

			if !found {
				return failed(s, t, fmt.Errorf("phase %q names effort %q, which engine %q does not offer", p.Name, p.Effort, p.Engine))
			}
		}

		if p.Thinking != "" && !eng.CanThink() {
			return failed(s, t, fmt.Errorf("phase %q configures thinking %q, but engine %q does not support thinking mode", p.Name, p.Thinking, p.Engine))
		}
	}

	wt, err := prepare(s, t)
	if err != nil {
		return failed(s, t, err)
	}

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

		notes := unconsumedNotes(s, t)
		if err := emit(s, t, phaseStart(p, i+1, notes)); err != nil {
			return err
		}

		inputPrev := ""
		if p.FeedOutput {
			inputPrev = prevOutput
		}

		var (
			streamErr                                             error
			streamedThoughts, streamedRefusals, streamedToolCalls int
		)

		resumeSess := lastSession(s, t, p.Engine, engines[p.Engine])

		out, runErr := engines[p.Engine].Run(ctx, engine.Request{
			Prompt:      prompt(t, p, notes, inputPrev),
			Model:       p.Model,
			Effort:      p.Effort,
			Thinking:    p.Thinking,
			Dir:         wt,
			Permissions: p.Permissions,
			Resume:      resumeSess,
			OnEvent: func(ev engine.StreamEvent) {
				switch ev.Type {
				case "thought":
					streamedThoughts++

					if err := emit(s, t, phaseThought(p.Name, i+1, ev.Thought)); err != nil && streamErr == nil {
						streamErr = err
					}
				case "tool_call":
					streamedToolCalls++

					if err := emit(s, t, phaseToolCall(p.Name, i+1, ev.ToolCall)); err != nil && streamErr == nil {
						streamErr = err
					}
				case "refusal":
					streamedRefusals++

					if err := emit(s, t, phaseRefused(p.Name, i+1, ev.Refusal)); err != nil && streamErr == nil {
						streamErr = err
					}
				}
			},
		})
		if streamErr != nil {
			return failed(s, t, fmt.Errorf("task %s, phase %q stream event emit: %w", t.ID, p.Name, streamErr))
		}

		if streamedThoughts == 0 {
			for _, th := range out.Thoughts {
				if err := emit(s, t, phaseThought(p.Name, i+1, th)); err != nil {
					return failed(s, t, fmt.Errorf("task %s, phase %q fallback thought emit: %w", t.ID, p.Name, err))
				}
			}
		}

		if streamedRefusals == 0 {
			for _, ref := range out.Refusals {
				if err := emit(s, t, phaseRefused(p.Name, i+1, ref)); err != nil {
					return failed(s, t, fmt.Errorf("task %s, phase %q fallback refusal emit: %w", t.ID, p.Name, err))
				}
			}
		}

		if streamedToolCalls == 0 {
			for _, tc := range out.ToolCalls {
				if err := emit(s, t, phaseToolCall(p.Name, i+1, tc)); err != nil {
					return failed(s, t, fmt.Errorf("task %s, phase %q fallback tool call emit: %w", t.ID, p.Name, err))
				}
			}
		}

		if runErr != nil {
			// A context that is done is not an engine that broke. The engine
			// reports being killed the same way it reports falling over —
			// exec gives it no other vocabulary — so the difference between
			// "the model failed" and "you stopped it" is not in the error at
			// all, it is here, and only this function can tell them apart.
			// Asking the context first is what keeps a task somebody
			// cancelled from being written down as a task that broke.
			//
			// The check is inside the error branch, not before it: a phase
			// that finished in the same instant the deadline passed did the
			// work, and calling that a cancellation would throw away a
			// finished phase to make the bookkeeping neat.
			if ctxErr := ctx.Err(); ctxErr != nil {
				return stopped(s, t, p.Name, out, ctxErr)
			}
			// The engine's error is what the caller needs; a failure to
			// record it must not replace or mask that error, so this emit is
			// best-effort and its own error is discarded, for the same
			// reason as the one in failed below.
			_ = emit(s, t, phaseEnd(record.PhaseFailed, p.Name, out, runErr)) //nolint:errcheck // deliberate: see above

			return failed(s, t, fmt.Errorf("task %s, phase %q: %w", t.ID, p.Name, runErr))
		}

		if err := runGates(ctx, s, t, p, i+1, wt, out); err != nil {
			return err
		}

		if err := emit(s, t, phaseEnd(record.PhaseFinished, p.Name, out, nil)); err != nil {
			return err
		}

		if out.Output != "" {
			prevOutput = out.Output
		}
	}

	return emit(s, t, record.Event{Kind: record.TaskFinished})
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
