package task

import (
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// lastSession returns the most recent session identifier this engine
// recorded for this task, and "" when there is none or the engine cannot
// resume.
//
// Whose session it was is the whole of it. A session id means nothing to a
// tool that did not issue it: a flow may name a different engine on each
// phase — Run validates them phase by phase, so that is a supported flow,
// not a mistake — and handing codex a claude session was either an error the
// user had to decipher or, worse, a silent fresh start reported as a resume.
//
// engineName is the name the flow used and the name the record holds, which
// is not necessarily what the engine calls itself: the registry is keyed by
// the flow's word, and phase.started writes that word down. Matching on the
// recorded one is what makes this agree with the log rather than with a
// second opinion about it.
//
// The walk goes forward rather than backward because the two facts live on
// two lines: phase.started names the engine, and the event that ends the
// phase carries the session. Only walking in the order they were written
// keeps them together.
func lastSession(s *store.Store, t Task, engineName string, eng engine.Engine) string {
	if s == nil || eng == nil || engineName == "" || !eng.CanResume() {
		return ""
	}

	events, err := Events(s, t)
	if err != nil {
		// Best-effort, and said out loud rather than hidden. A record that
		// will not read means this phase opens a fresh session instead of
		// resuming one: a worse run, not a broken one — the engine still
		// works, it has simply forgotten. Failing the whole run over it
		// would cost more than it saves. But falling through in silence is
		// how a phase came to lose everything the phase before it knew with
		// nothing anywhere saying why, so it goes in the log where the
		// person wondering can find it.
		logger.Warn("task/run", "%s: could not read the record to resume the %s session, so it starts a new one: %v", t.ID, engineName, err)

		return ""
	}

	var running, found string

	for _, e := range events {
		if e.Kind == record.PhaseStarted {
			running = e.Data["engine"]
		}

		if running != engineName {
			continue
		}

		if sess := strings.TrimSpace(e.Data["session"]); sess != "" {
			found = sess
		}
	}

	return found
}
