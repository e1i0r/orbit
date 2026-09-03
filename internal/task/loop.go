package task

// A loop of phases, going round until something verifiable says they can
// stop.

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

// runLoop walks the phases of a loop until every check passes, or until the
// loop runs out of the turns it was given.
//
// The check is a command and its exit code, and that is the whole of the
// design: a flow that asked the model whether its own work passed would be
// asking it to mark its own paper, and the answer to that is always yes.
//
// Each turn is told what the check said last time. Without that it is the
// same turn three times — the rule the attempt cap already keeps for one
// phase, kept here for a group of them — so the failure travels in the same
// shape a refused gate does, and lands in the prompt through the same
// section.
//
// The turns are counted in the record as they happen rather than summed at
// the end. A reader watching a loop go round wants to know it is going
// round; a count that only exists once it stops is a count for the post
// mortem.
func runLoop(ctx context.Context, s *store.Store, t Task, f flow.Flow, p flow.Phase, wt string, engines map[string]engine.Engine, others []string) error {
	l := p.Loop

	var tried []gateRefusal

	for turn := 1; ; turn++ {
		for i, inner := range l.Phases {
			out, err := attempts(ctx, phaseRun{
				store: s, task: t, flow: f, phase: inner, eng: engines[inner.Engine],
				n: i + 1, wt: wt, others: others, tried: tried,
			}, f.AttemptCap())
			if err != nil {
				return err
			}

			_ = out //nolint:wsl // the loop's checks are what judge the work, not what a phase printed
		}

		refused, err := runGates(ctx, s, t, checkPhase(p), turn, wt, engine.Result{})
		if err != nil {
			return err
		}

		if err := checked(s, t, p, turn, l.Max, refused); err != nil {
			return failed(s, t, err)
		}

		if refused == nil {
			return nil
		}

		if turn >= l.Max {
			return stopLooping(s, t, p, append(tried, *refused))
		}

		tried = append(tried, *refused)
	}
}

// checkPhase is the loop's checks as a phase for runGates to run.
//
// Its gates are the loop's Until, so one piece of code runs a command in a
// worktree and writes down what it answered — a second copy of that would
// be a second set of rules about what a failing command means.
func checkPhase(p flow.Phase) flow.Phase {
	return flow.Phase{Name: p.Name, Gates: p.Loop.Until}
}

// checked writes down what the turn answered.
func checked(s *store.Store, t Task, p flow.Phase, turn, max int, refused *gateRefusal) error {
	data := map[string]string{
		"loop":  p.Name,
		"turn":  strconv.Itoa(turn),
		"turns": strconv.Itoa(max),
	}

	if refused == nil {
		data["passed"] = "true"

		return emit(s, t, record.Event{Kind: record.LoopChecked, Phase: p.Name, Data: data})
	}

	data["passed"] = "false"
	data["check"] = refused.Gate
	data["exit"] = strconv.Itoa(refused.Exit)

	return emit(s, t, record.Event{Kind: record.LoopChecked, Phase: p.Name, Text: refused.Output, Data: data})
}

// stopLooping ends a run whose loop never went green.
//
// task.stuck and not task.failed, for the reason a phase out of attempts is
// stuck: nothing broke, and what is left is a decision. The text carries
// every turn, because the reader picking this up is being asked whether the
// check is wrong or the work is, and neither can be answered from the last
// failure alone.
func stopLooping(s *store.Store, t Task, p flow.Phase, tried []gateRefusal) error {
	var b strings.Builder

	fmt.Fprintf(&b, "The loop %q went round %d times and %q never passed.\n",
		p.Name, len(tried), tried[len(tried)-1].Gate)

	for i, ref := range tried {
		fmt.Fprintf(&b, "\nTurn %d — %q, exit %d:\n%s\n", i+1, ref.Gate, ref.Exit, lastLines(ref.Output, stuckLines))
	}

	text, _ := captured(b.String())
	_ = emit(s, t, record.Event{ //nolint:errcheck // best-effort: the run is ending either way
		Kind: record.TaskStuck,
		Text: text,
		Data: map[string]string{
			"attempts": strconv.Itoa(len(tried)),
			"phase":    p.Name,
			"gate":     tried[len(tried)-1].Gate,
		},
	})

	return fmt.Errorf("task %s: the loop %q went round %d times and %q never passed",
		t.ID, p.Name, len(tried), tried[len(tried)-1].Gate)
}
