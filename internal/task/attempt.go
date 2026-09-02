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
	store  *store.Store
	task   Task
	phase  flow.Phase
	eng    engine.Engine
	n      int      // which phase of the flow this is
	wt     string   // the worktree it runs in
	notes  []string // what the operator has said since the last phase
	prev   string   // what the phase before it said, when this one asked to be fed it
	others []string // the repositories it has not joined, by name
	tried  []gateRefusal
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

		if n >= allowed {
			return out, r.exhausted(out, *refused)
		}

		if err := r.retried(*refused, n, allowed); err != nil {
			return out, failed(r.store, r.task, err)
		}

		refused.Said = out.Output
		r.tried = append(r.tried, *refused)
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
func (r phaseRun) retried(ref gateRefusal, n, allowed int) error {
	return emit(r.store, r.task, record.Event{
		Kind:  record.PhaseRetried,
		Phase: r.phase.Name,
		Data: map[string]string{
			"gate":     ref.Gate,
			"exit":     strconv.Itoa(ref.Exit),
			"attempt":  strconv.Itoa(n),
			"attempts": strconv.Itoa(allowed),
		},
	})
}

// exhausted ends a phase no attempt could get past its gate.
//
// The error names the gate and not the number of attempts, because the gate
// is what a reader has to go and look at; how many times it was tried is in
// the record, once per phase.retried, and the window counts them there.
func (r phaseRun) exhausted(out engine.Result, ref gateRefusal) error {
	cause := fmt.Errorf("gate %q failed (exit %d)", ref.Gate, ref.Exit)
	_ = emit(r.store, r.task, phaseEnd(record.PhaseFailed, r.phase.Name, out, cause)) //nolint:errcheck // best-effort: see broke

	return failed(r.store, r.task, fmt.Errorf("task %s, phase %q: %w", r.task.ID, r.phase.Name, cause))
}
