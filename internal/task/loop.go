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
func runLoop(ctx context.Context, r loopRun) (engine.Result, error) {
	var (
		l     = r.phase.Loop
		s, t  = r.store, r.task
		p     = r.phase
		out   engine.Result
		tried []gateRefusal
	)

	for turn := 1; ; turn++ {
		// The reader is asked between turns as they are asked between
		// phases. A loop that went round twenty times without asking left
		// pause, cancel and skip unanswered for as long as it ran, which is
		// the one stretch of a run somebody is most likely to want to stop.
		decision, gateErr := ask(ctx, r.gate, t, p, turn)
		if gateErr != nil {
			return out, failed(s, t, fmt.Errorf("task %s, before turn %d of %q: %w", t.ID, turn, p.Name, gateErr))
		}

		switch decision {
		case Stop:
			return out, gateStop(s, t, p.Name, ctx.Err())
		case Skip:
			// Skipping a loop is leaving it. Half a turn is not a shape the
			// record can describe, so the block ends here and the flow goes
			// on to the phase after it.
			return out, nil
		}

		// Asked every turn and not only where the flow's phases meet: a
		// twenty-turn loop is where the money goes, and a cap checked only
		// on the way in is a cap a loop walks straight past.
		if spent, budget, over := overBudget(s, t); over {
			return out, stopSpending(s, t, p, spent, budget)
		}

		for i, inner := range l.Phases {
			one := phaseRun{
				store: s, task: t, phase: inner, eng: r.engines[inner.Engine],
				n: i + 1, wt: r.wt, others: r.others, tried: tried,
			}

			// What a person said goes to the first phase of the first turn
			// and nowhere else. It was taken before the loop began, and the
			// phase.started every inner phase emits is what marks it
			// consumed — so a loop that never took it swallowed it, and the
			// phase after the loop was told nothing either.
			if turn == 1 && i == 0 {
				one.notes, one.reviews = r.notes, r.reviews
			}

			var err error

			out, err = attempts(ctx, one, r.flow.AttemptCap())
			if err != nil {
				return out, err
			}
		}

		refused, err := runGates(ctx, s, t, checkPhase(p), turn, r.wt, engine.Result{})
		if err != nil {
			return out, err
		}

		if err := checked(s, t, p, turn, l.Max, refused); err != nil {
			return out, failed(s, t, err)
		}

		if refused == nil {
			return out, nil
		}

		if turn >= l.Max {
			return out, stopLooping(s, t, p, append(tried, *refused))
		}

		tried = append(tried, *refused)
	}
}

// loopRun is one loop and everything it needs to go round.
//
// A struct rather than eleven parameters, for the reason phaseRun is one: a
// call nobody can read is where the arguments start swapping places.
type loopRun struct {
	store   *store.Store
	task    Task
	flow    flow.Flow
	phase   flow.Phase
	wt      string
	engines map[string]engine.Engine
	others  []string
	notes   []string
	reviews []string
	gate    Gate
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
