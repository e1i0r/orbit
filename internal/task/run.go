package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"unicode/utf8"

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
	// task.started is written first — before the flow is validated, before
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

	// The run marker goes down next and comes off on every way out of here,
	// which is what lets any reader tell a phase still running from a phase
	// whose process is gone. It cannot survive SIGKILL — nothing written by
	// the dying process can — so the invariant a reader may rely on is the
	// weaker, true one: a task's log ends in a terminal event, or a reader
	// appends one. Reconcile is the reader that appends it.
	release, err := hold(s, t)
	if err != nil {
		return failed(s, t, err)
	}
	defer release()

	if err := f.Validate(); err != nil {
		return failed(s, t, err)
	}
	for _, p := range f.Phases {
		if _, ok := engines[p.Engine]; !ok {
			return failed(s, t, fmt.Errorf("phase %q wants the engine %q, which is not configured", p.Name, p.Engine))
		}
	}

	wt, err := prepare(s, t)
	if err != nil {
		return failed(s, t, err)
	}

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

		if err := emit(s, t, record.Event{
			Kind:  record.PhaseStarted,
			Phase: p.Name,
			Data:  map[string]string{"engine": p.Engine, "model": p.Model, "n": strconv.Itoa(i + 1)},
		}); err != nil {
			return err
		}

		out, runErr := engines[p.Engine].Run(ctx, engine.Request{
			Prompt: prompt(t, p),
			Model:  p.Model,
			Dir:    wt,
		})
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

		if err := emit(s, t, phaseEnd(record.PhaseFinished, p.Name, out, nil)); err != nil {
			return err
		}
	}

	return emit(s, t, record.Event{Kind: record.TaskFinished})
}

// phaseEnd is the one event that ends a phase, whichever way it ended.
//
// The three endings carry the same facts because the same things are true of
// all three: the engine printed something, it may have cost money, it may
// have a session somebody wants to resume. Only the kind differs, and cause
// — the reason a phase that did not finish stopped.
//
// That the ending carries the output at all is a fix. run.go bound the
// engine's answer and then threw it away on the error path, so every failed
// or cancelled phase lost everything the agent had printed before it died,
// which is the case where a reader most wants it: on a cancellation it is
// the only evidence of what the run did before it was stopped. Claude.Run
// returns its captured stdout alongside its error precisely so this can keep
// it (claude.go:45-51).
func phaseEnd(kind, phase string, out engine.Result, cause error) record.Event {
	text, full := captured(out.Output)
	e := record.Event{Kind: kind, Phase: phase, Text: text}
	data := map[string]string{}
	if full > 0 {
		data["output_bytes"] = strconv.Itoa(full)
	}
	if out.SessionID != "" {
		data["session"] = out.SessionID
	}
	if out.Cost != 0 {
		data["cost"] = strconv.FormatFloat(out.Cost, 'f', -1, 64)
	}
	if cause != nil {
		// Why it stopped goes in Data rather than Text, because Text is now
		// what the engine printed, and a log that ends at phase.failed — it
		// can, the write after it is best-effort — must still say why. It is
		// cut to the same length for the same reason: one event is one line
		// and record.MaxLine is what a line may weigh, and an engine's error
		// can carry the whole of its stderr.
		msg, _ := captured(cause.Error())
		data["error"] = msg
	}
	if len(data) > 0 {
		e.Data = data
	}
	return e
}

// stopped writes down that a run was stopped from outside rather than broken
// from inside, and hands back the context's error so the caller sees which.
//
// A cancellation and a timeout are different facts and the window bands them
// differently — you stopped this one, and this one outlived its deadline and
// wants you — so they are different kinds rather than one kind with a
// reason. Both are terminal: whichever wrote the phase, the attempt is over.
func stopped(s *store.Store, t Task, phase string, out engine.Result, cause error) error {
	// Best-effort, and the errors discarded, for the same reason as in
	// failed below: what stopped the run matters more than a failure to
	// write it down, and there is nobody left to hand a second error to.
	_ = emit(s, t, phaseEnd(record.PhaseCancelled, phase, out, nil)) //nolint:errcheck // deliberate: see failed
	kind := record.TaskCancelled
	if errors.Is(cause, context.DeadlineExceeded) {
		kind = record.TaskTimedOut
	}
	_ = emit(s, t, record.Event{Kind: kind}) //nolint:errcheck // deliberate: see failed
	return fmt.Errorf("task %s, phase %q: %w", t.ID, phase, cause)
}

// failed writes down that the run stopped and why, then hands the error back
// unchanged so the caller sees exactly what went wrong.
//
// Every way out of Run goes through here or through the phase-failure path
// above, because a run has four ways to fail and two of them used to return
// before anything was written: an invalid flow and an engine nobody
// configured both left a task that had no record at all, while a bad
// worktree left one that said task.failed. "Did this task fail?" has to be
// answerable from the log, since the log is the only thing the window will
// read.
//
// Recording is best-effort and its own error is discarded on purpose: a
// failure to write down why a run died must never replace the error that
// killed it.
func failed(s *store.Store, t Task, err error) error {
	// Cut for the same reason phaseEnd cuts: this text can be an engine's
	// error with the whole of its stderr inside it, and an event too large
	// to write is a failure nobody can read afterwards.
	text, _ := captured(err.Error())
	_ = emit(s, t, record.Event{Kind: record.TaskFailed, Text: text}) //nolint:errcheck // deliberate: see above
	return err
}

// maxOutput is how much of an engine's answer is kept in the record.
//
// One event is one line of JSON and record.MaxLine is what a line may weigh,
// so an unbounded stdout in Event.Text is a refused write waiting to happen
// — and a refused write is a phase that finished with nothing recorded. A
// megabyte is generous for a phase's last word. The design's home for the
// whole of it is phases/<n>/, which this plan does not build.
const maxOutput = 1 << 20

// captured cuts an engine's output down to what the record can hold and says
// in the text when it had to. Truncation that announces itself is honest;
// silent loss is not. The second return is the size of what was said, zero
// when nothing was cut.
func captured(out string) (text string, full int) {
	if len(out) <= maxOutput {
		return out, 0
	}
	n := maxOutput
	// Never cut a rune in half: the record is UTF-8, and a severed tail
	// would come back from the log as a replacement character.
	for n > 0 && !utf8.RuneStart(out[n]) {
		n--
	}
	return out[:n] + fmt.Sprintf("\n…[truncated, full output was %d bytes]", len(out)), len(out)
}

// prepare makes the worktree, reusing one that is already there so that a
// re-run picks up where the last one stopped rather than starting over.
func prepare(s *store.Store, t Task) (string, error) {
	wt, err := s.CreateWorktreeParent(t.Repo.Path, t.ID)
	if err != nil {
		return "", err
	}
	if _, statErr := os.Stat(wt); statErr == nil {
		return wt, nil
	}
	if err := t.Repo.AddWorktree(wt, "orbit/"+t.ID); err != nil {
		return "", err
	}
	return wt, nil
}

// prompt is what the engine is told for one phase.
//
// It is deliberately thin. Real prompts per phase, loaded from files and
// embedded in the binary, arrive with the rest of the flow catalogue; putting
// them here now would bury them in Go.
func prompt(t Task, p flow.Phase) string {
	return fmt.Sprintf("Phase: %s\nRepository: %s\n\nTask %s:\n%s\n", p.Name, t.Repo.Name, t.ID, t.Text)
}
