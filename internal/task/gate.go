package task

// The gate: the one place a run stops for a human, and the switch that
// decides whether it stops at all.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Go is what a gate says about the phase that is about to run.
type Go int

// The three answers.
//
// Continue is first so that it is the zero value: a Go nobody set is a phase
// that runs, which is the only safe default for a value that decides whether
// work happens.
const (
	Continue Go = iota // run it
	Skip               // do not run it, and record nothing for it
	Stop               // end the run here; no later phase runs
)

// Gate is asked before every phase, and it is the only thing that can hold a
// run.
//
// Before is asked about every phase and not only the ones whose Phase.Wait
// says so, which is the decision the rest of this file follows from.
// Autopilot is a switch the reader flips while a run is going, and a pause is
// a word written while a phase is already under way; a Flow copied by value
// into Run before either happened cannot carry news of them. So Phase.Wait
// stops being the switch and becomes the default answer, and the gate — which
// reads the settings file and the control word each time it is asked — is
// what knows the rest.
//
// The interface is declared here because this is where it is consumed. Run
// takes one and a nil one means never stop.
type Gate interface {
	Before(ctx context.Context, t Task, p flow.Phase, n int) (Go, error)
}

// The two reasons a phase can be stopped, as the record spells them.
// internal/view reads why to tell a task that needs you from a task the
// reader is holding, and they are different rows.
const (
	whyFlow   = "flow"   // the phase asked, and nothing moves until you answer
	whyPaused = "paused" // the reader asked, and the run is still theirs
)

// howAutopilot is what phase.resumed says when nobody typed anything: the
// switch was flipped and the gate let go on its own.
const howAutopilot = "autopilot"

// FileGate stops a run on a word in a file and on the autopilot switch, and
// reads both every time it is asked.
//
// It blocks in the run's own process rather than returning at the pause and
// starting again later. Returning is the tidier shape and it was refused for
// one reason: the engine's session dies with the process, and the session is
// what makes taking the keyboard mid-task possible at all. A gate that blocks
// keeps one session alive across the stop.
//
// poll is how long it waits between looks at the control file. It is a
// parameter rather than a constant so a test can have a gate that is patient
// on a fake clock and `orbit run` can have one second.
func FileGate(s *store.Store, poll time.Duration) Gate {
	return fileGate{store: s, poll: poll}
}

type fileGate struct {
	store *store.Store
	poll  time.Duration
}

// Before answers for one phase: take the word a reader left, and if there is
// none, ask the phase and the switch.
func (g fileGate) Before(ctx context.Context, t Task, p flow.Phase, _ int) (Go, error) {
	if ctx.Err() != nil {
		// A context that is done is a decision and not a fault: Stop is the
		// answer, and Run asks the context itself for which kind of stop it
		// was. Handing the context's error back here would make gateStop's
		// job someone else's and would report a cancellation as a gate that
		// broke.
		return Stop, nil //nolint:nilerr // deliberate: see above
	}
	word, err := take(g.store, t)
	if err != nil {
		return Continue, err
	}
	switch word {
	case wordCancel:
		return Stop, nil
	case wordSkip:
		return Skip, nil
	case wordResume, wordContinue:
		// Taken, and deliberately not acted on. A word that releases a wait
		// means nothing when no wait is in progress, and `orbit resume` on a
		// task no run holds writes exactly that word. Answering Continue here
		// would let it sit in the file until the next run and wave that run's
		// first gate through, with nothing in the record to say why a phase
		// whose flow asked to stop did not. Consuming it and falling through
		// to the ordinary decision is what makes a stale word harmless.
	}
	auto, err := autopilot(g.store)
	if err != nil {
		return Continue, err
	}
	// The flow's ask wins the label when both asked, because the two
	// mistakes are not equal: a held task shown as needing you is noise, and
	// a task that needs you shown as merely held is a task nobody comes back
	// to. The reader's pause is still remembered — it is what keeps
	// autopilot from lifting a brake the reader put on.
	switch {
	case p.Wait && !auto:
		return g.wait(ctx, t, p, whyFlow, word == wordPause)
	case word == wordPause:
		return g.wait(ctx, t, p, whyPaused, true)
	}
	return Continue, nil
}

// wait parks the run at a phase boundary and writes down that it is parked,
// which is what lets the window say "needs you" about a run it did not see
// stop and answer "how long has this been sitting on me?".
//
// byReader says the stop is the reader's own. Autopilot lifts the flow's
// gates and only those: flipping the switch on releases a phase its flow held,
// and never one a person pressed pause on — that one is theirs to lift.
func (g fileGate) wait(ctx context.Context, t Task, p flow.Phase, why string, byReader bool) (Go, error) {
	if err := emit(g.store, t, record.Event{
		Kind:  record.PhaseWaiting,
		Phase: p.Name,
		Data:  map[string]string{"why": why},
	}); err != nil {
		return Continue, err
	}
	for {
		word, err := take(g.store, t)
		if err != nil {
			return Continue, err
		}
		switch word {
		case wordCancel:
			// No phase.resumed: nothing was let go. The run ends where it
			// stands and Run writes the task-level event that says so.
			return Stop, nil
		case wordResume, wordContinue, wordSkip:
			if err := g.resumed(t, p, word); err != nil {
				return Continue, err
			}
			if word == wordSkip {
				return Skip, nil
			}
			return Continue, nil
		}
		if !byReader {
			auto, err := autopilot(g.store)
			if err != nil {
				return Continue, err
			}
			if auto {
				return Continue, g.resumed(t, p, howAutopilot)
			}
		}
		select {
		case <-ctx.Done():
			return Stop, nil
		case <-time.After(g.poll):
		}
	}
}

// resumed writes down that the phase was let go, and what let it go. The
// record has to answer that on its own: a reader coming back to a log wants
// to know whether a person released this or the switch did.
func (g fileGate) resumed(t Task, p flow.Phase, how string) error {
	return emit(g.store, t, record.Event{
		Kind:  record.PhaseResumed,
		Phase: p.Name,
		Data:  map[string]string{"how": how},
	})
}

// autopilot reads the switch. It is read afresh every time rather than once
// per run, which is the whole of what makes it live.
func autopilot(s *store.Store) (bool, error) {
	cfg, err := s.Settings()
	if err != nil {
		return false, err
	}
	return cfg.Autopilot, nil
}

// ask puts the question, and answers Continue when there is no gate at all.
//
// A nil gate means never stop. That is what keeps a caller who has no reader
// to release the run — a test, and `orbit run` before this existed — from
// having to supply a gate that says yes to everything.
func ask(ctx context.Context, g Gate, t Task, p flow.Phase, n int) (Go, error) {
	if g == nil {
		return Continue, nil
	}
	return g.Before(ctx, t, p, n)
}

// gateStop ends a run its gate would not let past, and says which kind of
// stop it was.
//
// A reader who wrote cancel, a SIGTERM and a deadline all arrive here, and
// the record tells them apart the way it does everywhere else: cancelled is
// something somebody chose and is done with, timed out is nobody's choice and
// wants you. Only the task-level event is written — no phase.cancelled —
// because no phase started: what the log holds is a phase.waiting naming the
// phase this run never reached.
//
// The write is best-effort and its error discarded for the same reason as in
// failed: what stopped the run matters more than a failure to write it down.
func gateStop(s *store.Store, t Task, phase string, ctxErr error) error {
	kind := record.TaskCancelled
	if errors.Is(ctxErr, context.DeadlineExceeded) {
		kind = record.TaskTimedOut
	}
	_ = emit(s, t, record.Event{Kind: kind}) //nolint:errcheck // deliberate: see failed
	if ctxErr != nil {
		return fmt.Errorf("task %s, waiting to start phase %q: %w", t.ID, phase, ctxErr)
	}
	return fmt.Errorf("task %s was stopped before phase %q", t.ID, phase)
}
