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
		// prev is the last thing said, which on the first turn is what the
		// phase before the loop said and after that what the turn before
		// said. A phase inside a loop reads feed_output exactly as a phase
		// outside one does: the coverage flow Orbit ships asks for it on
		// the phase that fixes what the checks caught, and nothing here
		// set it, so that phase was handed nothing.
		prev = r.prev
	)

	for turn := turnsSoFar(s, t, p.Name) + 1; ; turn++ {
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

		// And the size of the change, for the same reason and from the
		// second turn on: the first turn's change was measured by
		// whoever ran the phase before this block. A loop is where a
		// diff grows — three turns at a phase that writes is three times
		// what one turn writes — and a loop that ran past the budget and
		// then ran out of turns ended as stuck, with the number nobody
		// had checked never mentioned at all.
		if turn > 1 {
			if v := overDiff(s, t, r.flow); v != nil {
				return out, stopChanging(s, t, p, *v)
			}
		}

		for i, inner := range l.Phases {
			one := phaseRun{
				store: s, task: t, phase: inner, eng: r.engines[putTo(inner.Engine, r.on)],
				n: i + 1, wt: r.wt, others: r.others, tried: tried,
				prev: fedOutput(inner, prev),
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

			// Only when there is something to carry, as Run does it: a
			// phase that finished silently leaves the last real answer
			// standing rather than blanking it.
			if out.Output != "" {
				prev = out.Output
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
			return out, stopLooping(s, t, p, turn, append(tried, *refused))
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
	// on is the engine a relay has handed the task to, which outranks what
	// every inner phase names for the reason it does in Run: the flow was
	// written before anybody knew which engine would still have allowance.
	on string
	// prev is what the phase before the loop said, for an inner phase that
	// asked to be fed it. Untamed by fedOutput here: whether a phase is
	// fed at all is that phase's own flag, and the inner phases each have
	// one.
	prev    string
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

// turnsSoFar is how many turns this loop has already been round in this
// run, read from the record rather than from a counter.
//
// The counter was the loop's own, and an engine that ran out in the middle
// of one ends the call: the relay hands the task on and the loop is walked
// again from the top, with max counting from one a second time. A loop
// written as three turns went round three under each engine the machine
// had, paying for every one of them. The record is the one place that
// knows what has already happened, which is the same reason the money
// spent is read from it rather than added up in a variable.
//
// This run and not the last: counted from the newest task.started, so a
// task somebody started again gets its turns back.
func turnsSoFar(s *store.Store, t Task, phase string) int {
	events, err := Events(s, t)
	if err != nil {
		// A record that will not read is a loop that starts at one, which
		// is what it did before this was written. Refusing to go round
		// would stop a run over a number nobody can check.
		return 0
	}

	turns := 0

	for _, e := range events {
		switch {
		case e.Kind == record.TaskStarted:
			turns = 0
		case e.Kind == record.LoopChecked && e.Data["loop"] == phase:
			turns++
		}
	}

	return turns
}

// stopLooping ends a run whose loop never went green.
//
// task.stuck and not task.failed, for the reason a phase out of attempts is
// stuck: nothing broke, and what is left is a decision. The text carries
// every turn it has a refusal for, because the reader picking this up is
// being asked whether the check is wrong or the work is, and neither can be
// answered from the last failure alone.
//
// turn is what the record counted and len(tried) is what this call saw:
// they differ when the loop changed engine halfway, where the turns before
// the relay were another call's. The count a reader is given is the
// record's.
func stopLooping(s *store.Store, t Task, p flow.Phase, turn int, tried []gateRefusal) error {
	var b strings.Builder

	fmt.Fprintf(&b, "The loop %q went round %d times and %q never passed.\n",
		p.Name, turn, tried[len(tried)-1].Gate)

	for i, ref := range tried {
		first := turn - len(tried) + i + 1
		fmt.Fprintf(&b, "\nTurn %d — %q, exit %d:\n%s\n", first, ref.Gate, ref.Exit, lastLines(ref.Output, stuckLines))
	}

	text, _ := captured(b.String())
	_ = emit(s, t, record.Event{ //nolint:errcheck // best-effort: the run is ending either way
		Kind: record.TaskStuck,
		Text: text,
		Data: map[string]string{
			"attempts": strconv.Itoa(turn),
			"phase":    p.Name,
			"gate":     tried[len(tried)-1].Gate,
		},
	})

	return fmt.Errorf("task %s: the loop %q went round %d times and %q never passed",
		t.ID, p.Name, turn, tried[len(tried)-1].Gate)
}
