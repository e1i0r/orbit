package task

// One phase, run until its gates let it stand or the flow runs out of
// attempts.
//
// It lives beside run.go rather than in it because the two are different
// jobs: run.go walks a flow and decides what happens between phases, and
// this file is what happens inside one — the engine, the stream it writes
// down, the gates, and the decision to go round again. Keeping them apart is
// also what keeps either under the size ceiling.

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// stuckLines is how much of each gate's output the stuck summary repeats.
// Enough for a failing test's own report, short enough that three of them
// are still one thing a person reads.
const stuckLines = 20

// gateRefusal is a gate that said no, and everything the next attempt is
// owed about it: which gate, what it returned, what it printed, and what the
// engine had answered when it was run past it.
type gateRefusal struct {
	Gate   string
	Exit   int
	Output string
	Said   string
}

// refusals is the section of the prompt that says what the attempts before
// this one tried and why the gate would not have it.
//
// All of them rather than only the last, because the failure this whole
// mechanism exists to avoid is three identical shots at the same wall: an
// attempt that can read that the symbol was already renamed once, and that
// the gate refused it anyway, is the only one with a reason to look
// somewhere else. What the engine answered is quoted beside what the gate
// printed for the same reason — "what failed" without "what was tried" is
// half of the sentence.
func refusals(tried []gateRefusal) string {
	if len(tried) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n## Previous attempts at this phase\n\n" +
		"Each of these ran in this same worktree and was refused by the gate " +
		"named. What they left behind is still here; what they tried did not " +
		"pass, so do not try it again.\n")

	for i, r := range tried {
		fmt.Fprintf(&b, "\n### Attempt %d — gate `%s` refused it (exit %d)\n", i+1, r.Gate, r.Exit)

		if r.Said != "" {
			fmt.Fprintf(&b, "\nWhat that attempt answered:\n\n%s\n", engine.Fenced(r.Said))
		}

		fmt.Fprintf(&b, "\nWhat the gate printed:\n\n%s\n", engine.Fenced(r.Output))
	}

	return b.String()
}

// phaseRun is one phase and everything it needs to be run, gathered once so
// that running it again costs nothing but the engine.
type phaseRun struct {
	store *store.Store
	task  Task
	// flow and n place the phase in its own flow, which is what says
	// whether it is the last one — and the last one is the only phase that
	// can tell how the task ended.
	flow  flow.Flow
	phase flow.Phase
	eng   engine.Engine
	n     int      // which phase of the flow this is
	wt    string   // the worktree it runs in
	notes []string // what the operator has said since the last phase
	// reviews is what people said on the pull request and no phase has
	// answered yet. It is carried beside the notes because it is the same
	// kind of thing — an instruction from a person about this task — and
	// read after them, because the operator is who settles a disagreement
	// between the two.
	reviews []string
	prev    string   // what the phase before it said, when this one asked to be fed it
	others  []string // the repositories it has not joined, by name
	tried   []gateRefusal
}

// attempts runs one phase until a gate lets it stand or there are no
// attempts left.
//
// The cap counts the first run, so a flow that allows three attempts runs a
// phase three times and not four. Each attempt after the first is told what
// the ones before it tried and why the gate refused them — without that they
// are three identical shots at the same wall, which is what a retry loop is
// worth nothing as.
func attempts(ctx context.Context, r phaseRun, allowed int) (engine.Result, error) {
	for n := 1; ; n++ {
		out, refused, err := r.once(ctx)
		if err != nil {
			return out, err
		}

		if refused == nil {
			return out, nil
		}

		refused.Said = out.Output
		r.tried = append(r.tried, *refused)

		if n >= allowed {
			return out, r.exhausted(out, n)
		}

		if err := r.retried(out, *refused, n, allowed); err != nil {
			return out, failed(r.store, r.task, err)
		}
	}
}

// once is a single run of the phase: the engine, whatever it streamed, and
// the gates that judge what it left behind.
//
// A gate that refused comes back as a value rather than as an error, because
// it is not one yet: whether a refusal ends the run is the caller's decision
// and depends on how many attempts are left. An error here is a run that is
// already over — the terminal event is written by the time it is returned.
func (r phaseRun) once(ctx context.Context) (engine.Result, *gateRefusal, error) {
	// Through failed, and not returned bare. Every other way out of this
	// function writes a terminal event, and the reason is the invariant
	// hold's comment states in run.go: a task's log ends in a terminal
	// event, or a reader appends one. Reconcile is that reader, and it
	// cannot be here — it acts on a stale marker, and the marker of this run
	// is released cleanly when Run returns. So an emit that failed would
	// leave task.started with nothing after it: a task that reads as running
	// for ever, in every reader of the record.
	if err := emit(r.store, r.task, phaseStart(r.phase, r.n, r.notes)); err != nil {
		return engine.Result{}, nil, failed(r.store, r.task, err)
	}

	out, runErr, err := r.run(ctx)
	if err != nil {
		return out, nil, err
	}

	if runErr != nil {
		return out, nil, r.broke(ctx, out, runErr)
	}

	refused, err := runGates(ctx, r.store, r.task, r.phase, r.n, r.wt, out)
	if err != nil || refused != nil {
		return out, refused, err
	}

	if err := emit(r.store, r.task, phaseEnd(record.PhaseFinished, r.phase.Name, out, nil)); err != nil {
		return out, nil, failed(r.store, r.task, err)
	}

	return out, nil, nil
}

// broke ends a run whose engine did not come back.
//
// A context that is done is not an engine that broke. The engine reports
// being killed the same way it reports falling over — exec gives it no other
// vocabulary — so the difference between "the model failed" and "you stopped
// it" is not in the error at all, it is here. Asking the context first is
// what keeps a task somebody cancelled from being written down as a task
// that broke.
//
// It is asked on the error path only: a phase that finished in the same
// instant the deadline passed did the work, and calling that a cancellation
// would throw away a finished phase to make the bookkeeping neat.
func (r phaseRun) broke(ctx context.Context, out engine.Result, runErr error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return stopped(r.store, r.task, r.phase.Name, out, ctxErr)
	}
	// The engine's error is what the caller needs; a failure to record it
	// must not replace or mask that error, so this emit is best-effort and
	// its own error is discarded, for the same reason as the one in failed.
	_ = emit(r.store, r.task, phaseEnd(record.PhaseFailed, r.phase.Name, out, runErr)) //nolint:errcheck // deliberate: see above

	return failed(r.store, r.task, fmt.Errorf("task %s, phase %q: %w", r.task.ID, r.phase.Name, runErr))
}

// retried writes down the seam between one attempt and the next.
//
// It is built by phaseEnd, like the three events that end a phase for good,
// because it ends one too: this is the only line the record will ever hold
// about the attempt a gate refused, and an attempt that ran was paid for.
// Written any other way, what two refused attempts spent is money nothing
// adds up — the cost on the board would be the price of the attempt that
// happened to pass.
func (r phaseRun) retried(out engine.Result, ref gateRefusal, n, allowed int) error {
	e := phaseEnd(record.PhaseRetried, r.phase.Name, out, nil)
	if e.Data == nil {
		e.Data = map[string]string{}
	}

	e.Data["gate"] = ref.Gate
	e.Data["exit"] = strconv.Itoa(ref.Exit)
	e.Data["attempt"] = strconv.Itoa(n)
	e.Data["attempts"] = strconv.Itoa(allowed)

	return emit(r.store, r.task, e)
}

// exhausted ends a phase no attempt could get past its gate.
//
// The task is stuck and not failed. A failure is one run, and a run is what
// a reader answers by starting another one — which is exactly what has been
// happening here, twice already. task.stuck is the word for the end of
// retrying, and the fold reads it as a band of its own: nothing moves until
// somebody decides something.
//
// The summary of the attempts goes on the event rather than being left to
// be reassembled from the log, because the reader who needs it is the one
// opening the supervisor's thread, and reconstructing three attempts from a
// hundred lines of stream is the work this saves.
func (r phaseRun) exhausted(out engine.Result, spent int) error {
	last := r.tried[len(r.tried)-1]
	cause := fmt.Errorf("gate %q failed (exit %d)", last.Gate, last.Exit)
	_ = emit(r.store, r.task, phaseEnd(record.PhaseFailed, r.phase.Name, out, cause)) //nolint:errcheck // best-effort: see broke

	text, _ := captured(stuckLine(r.phase.Name, spent, r.tried))
	_ = emit(r.store, r.task, record.Event{ //nolint:errcheck // best-effort: see broke
		Kind: record.TaskStuck,
		Text: text,
		Data: map[string]string{
			"attempts": strconv.Itoa(spent),
			"phase":    r.phase.Name,
			"gate":     last.Gate,
		},
	})

	return fmt.Errorf("task %s, phase %q: %w, and the %d attempts it was allowed are spent",
		r.task.ID, r.phase.Name, cause, spent)
}

// stuckLine is what a human reads about a task that ran out of attempts: one
// sentence saying where it stopped, and then each attempt with the tail of
// what its gate printed.
//
// The tail rather than the head: a build says what is wrong at the end, and
// the first twenty lines of a test run are the tests that passed.
func stuckLine(phase string, spent int, tried []gateRefusal) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%d attempts at phase %q, and the gate %q refused every one of them.\n",
		spent, phase, tried[len(tried)-1].Gate)

	for i, ref := range tried {
		fmt.Fprintf(&b, "\nAttempt %d — gate %q, exit %d:\n%s\n",
			i+1, ref.Gate, ref.Exit, lastLines(ref.Output, stuckLines))
	}

	return b.String()
}

// lastLines is the final n lines of what a gate printed, and says so when it
// left anything out.
func lastLines(out string, n int) string {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) <= n {
		return out
	}

	return fmt.Sprintf("…[the first %d lines are not repeated here]\n%s",
		len(lines)-n, strings.Join(lines[len(lines)-n:], "\n"))
}
