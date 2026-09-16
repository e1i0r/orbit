package task

// Changing engine in the middle of a task, and carrying on.
//
// An engine that runs out of allowance has not broken anything. The work is
// in the worktree, the record says how far it got, and there are three other
// engines on the machine — so a run that ends there ends for a reason that
// has nothing to do with the task. This is what happens instead: the task is
// handed to an engine that has something left, told what the one before it
// got as far as doing, and carries on.
//
// Whether it happens without asking is autopilot's decision and not a switch
// of its own. Autopilot already means exactly this — whether a run walks its
// flow without stopping for a person — and a second question about the same
// decision is a second place to answer it differently.

import (
	"errors"
	"sort"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Spare is what one engine has left to spend, as far as anything can tell.
type Spare struct {
	// Free is whether it is worth handing work to.
	//
	// An engine nothing can read the allowance of is free. "Nobody can read
	// opencode's quota" and "opencode has none left" are different
	// sentences and only one of them is a reason not to try — a relay that
	// confused them would refuse to hand work to three of the four engines
	// on this machine, none of which has ever reported a number.
	Free bool
	// Back is how long until the window comes back, and zero when it is
	// free or when nothing knows. Zero is therefore not "it is back now":
	// it is "there is no hour to tell you", and what reads this says so
	// rather than printing a time it made up.
	Back time.Duration
}

// Allowance is how a run finds out what an engine has left.
//
// A port, because this package cannot see internal/quota and should not: what
// an allowance is depends on a proxy, a file of rollouts and an API key in
// the environment, none of which is a fact about running a task. A nil
// Allowance answers that every engine is free, which is what every run had
// before relays existed.
type Allowance func(name string) Spare

// callsItself is what an engine calls itself, and nothing for one that is not
// there. A phase with no engine cannot have run out, so the empty name never
// reaches a relay — it is here so that reading the name is not a place a run
// can panic.
func callsItself(e engine.Engine) string {
	if e == nil {
		return ""
	}

	return e.Name()
}

// spare is one engine's reading through a port that may not be there.
func spare(left Allowance, name string) Spare {
	if left == nil {
		return Spare{Free: true}
	}

	return left(name)
}

// noneLeft is a phase that ended because the engine holding it had nothing
// left to spend.
//
// An error and not a result, because a phase that ran out did not do the
// work — but its own kind of error, because it is the one failure another
// engine can answer. Everything else a phase can die of, the next engine
// would die of too: a compile error is a compile error whoever is typing.
type noneLeft struct {
	engine string
	err    error
}

func (n *noneLeft) Error() string { return n.err.Error() }
func (n *noneLeft) Unwrap() error { return n.err }

// ranDry is the ran-out error inside err, and nil when it is any other
// failure.
func ranDry(err error) *noneLeft {
	var none *noneLeft
	if errors.As(err, &none) {
		return none
	}

	return nil
}

// passTo decides what happens to a phase whose engine ran out: another
// engine takes it, or the run stops and says which of the two reasons it
// stopped for.
//
// The name it answers with is the engine to carry on under. An error is the
// run ending, and it is already written down by the time it is handed back.
func passTo(s *store.Store, t Task, p flow.Phase, from string,
	engines map[string]engine.Engine, tried []string, left Allowance,
) (string, error) {
	// A copy, because append on the caller's slice can write into its spare
	// capacity: whoever passed tried would find a name in it that this
	// function put there.
	asked := make([]string, 0, len(tried)+1)
	asked = append(asked, tried...)
	asked = append(asked, from)

	free, back := whoIsFree(s, engines, asked, left)

	if len(free) == 0 {
		return "", noEngine(s, t, p, from, back)
	}

	on, err := autopilot(s)
	if err != nil {
		// A settings file that will not answer is not a reason to spend
		// another engine's allowance without being asked. Asking is the
		// safe half of this decision, so an unreadable switch reads as off.
		logger.Error("task/relay", "read the autopilot switch for task %q: %v", t.ID, err)

		on = false
	}

	if !on {
		return "", needsEngine(s, t, p, from, free)
	}

	took := free[0]
	if err := emit(s, t, relayed(p.Name, from, took)); err != nil {
		return "", failed(s, t, err)
	}

	return took, nil
}

// whoIsFree is every engine with something left, in the order a relay should
// offer them, and how long until the first allowance comes back when none
// has anything.
//
// The engines already asked are excluded rather than re-read. An engine that
// ran out a minute ago may still read as free — a proxy caches, a rollout
// file is as fresh as the last run — and handing it the same phase again is
// the one way this turns into a loop.
func whoIsFree(s *store.Store, engines map[string]engine.Engine,
	asked []string, left Allowance,
) (free []string, back time.Duration) {
	for _, name := range inOrder(s, engines) {
		if contains(asked, name) {
			continue
		}

		reading := spare(left, name)
		if reading.Free {
			free = append(free, name)

			continue
		}

		if reading.Back > 0 && (back == 0 || reading.Back < back) {
			back = reading.Back
		}
	}

	return free, back
}

// inOrder is every engine this build knows, the reader's own first.
//
// Their own first because that is the only preference Orbit has been told:
// the settings name one engine, and it is the one a relay should reach for
// before any other. The rest go in the order their names sort in, which is
// nobody's preference but is the same order every time — a relay that picked
// differently on two runs of the same task would be a relay nobody could
// reason about.
func inOrder(s *store.Store, engines map[string]engine.Engine) []string {
	rest := make([]string, 0, len(engines))
	for name := range engines {
		rest = append(rest, name)
	}

	sort.Strings(rest)

	cfg, err := s.Settings()
	if err != nil || cfg.Engine == "" {
		return rest
	}

	out := make([]string, 0, len(rest))
	if _, known := engines[cfg.Engine]; known {
		out = append(out, cfg.Engine)
	}

	for _, name := range rest {
		if name != cfg.Engine {
			out = append(out, name)
		}
	}

	return out
}

// contains is whether a name is in a list, which is all this needs of a set.
func contains(all []string, one string) bool {
	for _, name := range all {
		if name == one {
			return true
		}
	}

	return false
}

// HandedTo writes down that a person moved a task to another engine.
//
// `orbit task start -engine` could already do this and nothing said so: the
// run that followed read as an ordinary run that happened to be on codex,
// and the reason it was on codex — that claude had run out an hour earlier —
// existed nowhere. A task that passed through three engines has to say so,
// or its result cannot be judged.
//
// It writes nothing when the task was already on that engine, and nothing
// when the record has never seen it run: the first run of a task is not a
// relay, it is a start.
func HandedTo(s *store.Store, t Task, to string) error {
	if to == "" {
		return nil
	}

	from, phase := lastOn(s, t)
	if from == "" || from == to {
		return nil
	}

	return emit(s, t, chose(phase, from, to))
}

// lastOn is the engine the record last saw running this task, and the phase
// it was running.
func lastOn(s *store.Store, t Task) (engine, phase string) {
	events, err := Events(s, t)
	if err != nil {
		logger.Error("task/relay", "read which engine last had task %q: %v", t.ID, err)
		return "", ""
	}

	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Kind == record.PhaseStarted {
			return events[i].Data["engine"], events[i].Phase
		}
	}

	return "", ""
}
