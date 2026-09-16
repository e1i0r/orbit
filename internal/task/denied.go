package task

// A phase that was refused what it needed, and left nothing behind.
//
// A headless run has nobody to ask, so a tool the posture does not grant is
// denied without a word. The engine handles it, writes a sentence saying it
// could not, and exits zero — and read by its exit code alone that is a
// success. A task that did nothing sat in done, where nobody looks again.
//
// It is its own ending because it sends a reader somewhere neither of the
// others does. A phase that broke is a bug to go and look at, one that ran
// out is a wait, and this one is a posture too narrow for the work it was
// given.

import (
	"fmt"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/record"
)

// denied is the tool a phase was refused when that is the whole story of the
// phase, and nothing when it is not.
//
// Both halves have to hold, and both have to be askable. A refusal on its
// own is ordinary — a phase that was turned down once and went another way
// did the work — and an empty worktree on its own is ordinary too, because a
// plan and a review are phases that write nothing by design. What is not
// ordinary is a phase that was denied something and then left nothing
// behind: that is a posture too narrow for the work, wearing the exit code
// of a success.
func (r phaseRun) denied(out engine.Result) string {
	if len(out.Refusals) == 0 {
		return ""
	}

	// A task against no repository has no tree to have left anything in, so
	// the second half of the rule cannot be asked. Saying nothing is the
	// only honest answer: this is a verdict built on two facts, and one of
	// them is unavailable.
	if r.task.Repo.Path == "" || r.wt == "" {
		return ""
	}

	if len(nowHeld(r.task, r.wt)) > 0 {
		return ""
	}

	for _, one := range out.Refusals {
		if one.Tool != "" {
			return one.Tool
		}
	}

	// An engine that reported a refusal without naming the tool still
	// reported one. Saying that something was denied is the half a reader
	// acts on; which tool is one line of `orbit task show` away.
	return unnamedTool
}

// unnamedTool stands for a refusal that arrived without a name on it.
const unnamedTool = "a tool"

// wasDenied ends a phase that was refused what it needed.
//
// It ends the run, because the next phase would be run under the same
// posture and denied the same thing. What a reader is sent to do is look at
// what the phase was allowed — not to wait, and not to debug.
func (r phaseRun) wasDenied(out engine.Result, tool string) error {
	e := phaseEnd(record.PhaseDenied, r.phase.Name, out, nil)
	if e.Data == nil {
		e.Data = map[string]string{}
	}

	e.Data["tool"] = tool

	//nolint:errcheck // best-effort: the caller's error is what matters, see failed
	_ = emit(r.store, r.task, e)

	return failed(r.store, r.task, fmt.Errorf(
		"task %s, phase %q was refused %s and left nothing behind: check what the phase is allowed",
		r.task.ID, r.phase.Name, tool))
}
