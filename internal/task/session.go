package task

import (
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// lastSession returns the most recent session identifier recorded for this task
// if the engine can resume sessions, and "" otherwise.
func lastSession(s *store.Store, t Task, eng engine.Engine) string {
	if s == nil || eng == nil || !eng.CanResume() {
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
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if sess := strings.TrimSpace(e.Data["session"]); sess != "" {
			return sess
		}
	}
	return ""
}
