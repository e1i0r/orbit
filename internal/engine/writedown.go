package engine

// What Orbit ran, and what came back.
//
// One place, because there is one place a program is started: spec.run is
// the single copy of what all three adapters do, so a fourth engine added
// tomorrow is written down on the day it is added, rather than on the day
// somebody notices it never was.
//
// An engine is the slowest, most expensive and least predictable thing this
// program starts, and the record says almost nothing about the running of
// one. phase.started and phase.finished are written by internal/task around
// the call, so the record knows a phase happened; it does not know that it
// took nineteen minutes, cost $2.10, and was the binary under the reader's
// home directory rather than the one on PATH. Those are the three things
// somebody asks about a run that went wrong, and none of them were written
// down anywhere.

import (
	"time"

	"github.com/e1i0r/orbit/internal/logger"
)

// noteStart writes down the process about to be started, and answers with
// the moment it began — which is what noteEnd is handed, so the clock for a
// run lives in this file instead of in the middle of run.
//
// The command line is described rather than printed. Every engine's argv
// ends with the prompt, a prompt is a page of somebody's prose, and a page
// of prose in a diagnostic log is a paragraph sitting between two facts.
// Its size is here because a prompt that grew to a megabyte is the kind of
// thing that explains a bill.
func noteStart(name, bin string, req Request) time.Time {
	logger.Info("engine/"+name,
		"running %s in %q: model=%q effort=%q thinking=%q permissions=%v resume=%t prompt=%dB",
		bin, req.Dir, req.Model, req.Effort, req.Thinking, req.Permissions, req.Resume != "", len(req.Prompt))

	return time.Now()
}

// noteEnd writes down what came back, and how long the wait for it was.
//
// Counted rather than quoted: what the engine answered is the answer to a
// phase and the record already holds it in full, so a second copy here
// would be the largest line in the file by an order of magnitude and would
// tell a reader nothing the record does not.
func noteEnd(name string, start time.Time, res Result, err error) {
	took := time.Since(start).Round(time.Millisecond)

	if err != nil {
		logger.Error("engine/"+name, "failed after %s: %v", took, err)

		return
	}

	logger.Info("engine/"+name,
		"answered after %s: session=%q cost=%.4f output=%dB thoughts=%d tools=%d refusals=%d",
		took, res.SessionID, res.Cost, len(res.Output), len(res.Thoughts), len(res.ToolCalls), len(res.Refusals))
}

// notStarted writes down a run that never began, and answers with the error
// it was given so that the two places this happens stay one line each.
//
// It says something different from noteEnd's failure, because it is a
// different fact: a command line that could not be built, or a binary that
// is not installed, is Orbit refusing in a millisecond and before anything
// was spent. "failed after 0s" would read as a model that was asked and
// said no.
func notStarted(name string, err error) error {
	logger.Error("engine/"+name, "did not start: %v", err)

	return err
}
