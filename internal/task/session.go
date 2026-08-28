package task

import (
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
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

	path, err := s.EventsPath(t.Repo.Path, t.ID)
	if err != nil {
		return ""
	}

	events, err := record.Read(path)
	if err != nil {
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
