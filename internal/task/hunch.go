package task

// The decision engine at a gate: whether a run that stopped has to wait for
// a person, or whether something that answers in half a second can say it is
// fine and let it go.
//
// This is where the time is. A run's engines work for minutes and its task
// takes days, and the difference is a phase sitting at a gate with "make
// check is green" as the last thing it said, waiting for somebody to read
// it. Nothing about that wait is the engine being slow.
//
// Three things keep it honest. It is asked only where the flow itself
// stopped the run, never where a person pressed pause — that is the rule
// autopilot already keeps, and a switch that could lift a reader's own
// brake is a switch nobody can use. It acts only on "done", and only above
// a floor the reader sets: "again" and "human" both end in a person, and
// the engine saying them changes nothing yet. And it writes down what it
// said whether or not it was acted on, because a supervisor that decided
// quietly is one no reader can check — in shadow, writing it down is the
// whole of what it does.

import (
	"context"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/hunch"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// howDecided is what phase.resumed says when the decision engine let a
// phase go: a reader coming back to the log has to be able to tell that
// from a person and from the autopilot switch.
const howDecided = "decided"

// lets asks the decision engine about a phase that has just stopped, and
// answers whether the gate may let it go.
//
// False for everything uncertain: no port, no key, the setting off, a
// model that could not be reached, a verdict below the floor, or any
// verdict that is not "done". The run then waits exactly as it did before
// there was a decision engine, which is what makes this safe to leave on.
func (g fileGate) lets(ctx context.Context, t Task, p flow.Phase) bool {
	verdict, cfg, ok := g.asked(ctx, t, p)
	if !ok {
		return false
	}

	acts := cfg.Deciding() == store.DecisionsOn &&
		verdict.Choice == hunch.Done &&
		verdict.Acts(cfg.DecisionBar())

	if err := emit(g.store, t, decisionEvent(p, verdict, cfg.Deciding(), acts)); err != nil {
		// Written before it is acted on, and not acted on if it could not
		// be written: a phase let go by a decision the record does not
		// hold is a run nobody can explain afterwards.
		logger.Error("task/decide", "%s: the decision about phase %q could not be written: %v",
			t.ID, p.Name, err)

		return false
	}

	return acts
}

// holds is the other direction, and the one autopilot asks for: a run that
// would have been waved through, stopped because something read what it did
// and is sure it is not finished.
//
// Only "again" and "human", and only above the floor. Everything else — a
// verdict of done, a model that is unsure, no key, no answer — is
// autopilot exactly as it was, which is what keeps this from becoming a
// switch that interrupts more than it settles.
//
// The reason it gives is the flow's own: what a reader sees is a phase that
// stopped for them, and why it stopped is the sentence in the record beside
// the decision.
func (g fileGate) holds(ctx context.Context, t Task, p flow.Phase) (string, bool) {
	verdict, cfg, ok := g.asked(ctx, t, p)
	if !ok {
		return "", false
	}

	hold := cfg.Deciding() == store.DecisionsOn &&
		(verdict.Choice == hunch.Again ||
			verdict.Choice == hunch.Human) &&
		verdict.Acts(cfg.DecisionBar())

	if err := emit(g.store, t, decisionEvent(p, verdict, cfg.Deciding(), hold)); err != nil {
		logger.Error("task/decide", "%s: the decision about phase %q could not be written: %v",
			t.ID, p.Name, err)

		return "", false
	}

	return whyFlow, hold
}

// asked puts the question, and answers false for every way there is not to
// have one: no port, no key, the setting off, or a model that could not be
// reached. Both callers read that the same way — the run goes on being
// whatever it was without a decision engine.
func (g fileGate) asked(
	ctx context.Context, t Task, p flow.Phase,
) (hunch.Verdict, store.Settings, bool) {
	if g.decider == nil {
		return hunch.Verdict{}, store.Settings{}, false
	}

	cfg, err := g.store.Settings()
	if err != nil || cfg.Deciding() == store.DecisionsOff {
		return hunch.Verdict{}, store.Settings{}, false
	}

	verdict, err := g.decider.Decide(ctx, hunch.Stop{
		Task:  t.ID,
		Asked: t.Text,
		Phase: p.Name,
		Said:  lastSaid(g.store, t),
	})
	if err != nil {
		// A decision that did not arrive is not a decision. The run goes
		// on as it would have, and the reason is worth a line in the log
		// rather than in the record: the record is the account of the
		// task, and an outage at somebody else's API is not something that
		// happened to the task.
		logger.Warn("task/decide", "%s: no decision about phase %q: %v", t.ID, p.Name, err)

		return hunch.Verdict{}, cfg, false
	}

	return verdict, cfg, true
}

// decisionEvent is what the record keeps of one decision.
//
// Every field a reader needs to argue with it: which of the three words,
// how sure, which model said so, what Orbit was allowed to do with it, and
// whether it did. In shadow the last two are the interesting ones — the
// answer is there to be compared with what the reader went on to do.
func decisionEvent(p flow.Phase, v hunch.Verdict, mode string, acted bool) record.Event {
	return record.Event{
		Kind:  record.Decided,
		Phase: p.Name,
		Data: map[string]string{
			"choice":     string(v.Choice),
			"confidence": strconv.FormatFloat(v.Confidence, 'f', 2, 64),
			"by":         v.Model,
			"mode":       mode,
			"acted":      strconv.FormatBool(acted),
		},
	}
}

// lastSaid is the last thing the run said before it stopped, which is what
// a supervisor reads.
//
// This run and not the last one, for the reason planText reads from the
// newest task.started: a verdict about what the previous attempt printed is
// a verdict about work that has already been replaced.
//
// Unless the run is a retry from a phase. It runs none of the phases before
// that one, so what they said is the work it stands on, and forgetting it
// asked the gate about a run that had said nothing: ORB-121 was answered
// "again" ten times over an implement that had reported make check green.
func lastSaid(s *store.Store, t Task) string {
	events, err := Events(s, t)
	if err != nil {
		return ""
	}

	said := ""

	for _, e := range events {
		switch e.Kind {
		case record.TaskStarted:
			if e.Data["from"] == "" {
				said = ""
			}
		case record.PhaseFinished, record.PhaseFailed, record.GateFailed:
			if e.Text != "" {
				said = e.Text
			}
		}
	}

	return said
}

// attempt is what task.started says about the attempt it begins: the flow,
// and the phase a retry began at. from is what lets lastSaid tell a retry
// that keeps the phases before it from a run that replaces them.
func attempt(flowName, from string) map[string]string {
	data := map[string]string{"flow": flowName}
	if name := strings.TrimSpace(from); name != "" {
		data["from"] = name
	}

	return data
}
