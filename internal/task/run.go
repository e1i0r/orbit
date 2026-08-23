package task

import (
	"context"
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
// It stops at the first failure and says why, in the words the engine used.
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
func Run(ctx context.Context, s *store.Store, t Task, f flow.Flow, engines map[string]engine.Engine) error {
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
	if err := emit(s, t, record.Event{Kind: "task.started", Data: map[string]string{"flow": f.Name, "worktree": wt}}); err != nil {
		return err
	}

	for i, p := range f.Phases {
		if err := emit(s, t, record.Event{
			Kind:  "phase.started",
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
			// The engine's error is what the caller needs; a failure to
			// record it must not replace or mask that error, so this emit is
			// best-effort and its own error is discarded, for the same
			// reason as the one in failed below.
			_ = emit(s, t, record.Event{Kind: "phase.failed", Phase: p.Name, Text: runErr.Error()}) //nolint:errcheck // deliberate: see above
			return failed(s, t, fmt.Errorf("task %s, phase %q: %w", t.ID, p.Name, runErr))
		}

		text, full := captured(out.Output)
		finished := record.Event{Kind: "phase.finished", Phase: p.Name, Text: text}
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
		if len(data) > 0 {
			finished.Data = data
		}
		if err := emit(s, t, finished); err != nil {
			return err
		}
	}

	return emit(s, t, record.Event{Kind: "task.finished"})
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
	_ = emit(s, t, record.Event{Kind: "task.failed", Text: err.Error()}) //nolint:errcheck // deliberate: see above
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
