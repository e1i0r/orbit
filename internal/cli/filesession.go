package cli

// Bringing back what was said in an interactive session.
//
// open.go hands the terminal over and writes down that it did. This is the
// other end of that gesture: the window comes back from a session it saw
// none of, and the conversation the reader had is on disk in the engine's
// transcript and nowhere in the record. Elio asked for it to belong to the
// task it was had on — "todo lo que hable por alla, pertenece a esa tarea" —
// so every turn of it is written into that task, both sides of it, however
// many there are.

import (
	"time"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
)

// fileSessionPort writes those turns down and answers how many it wrote.
//
// The directory is worked out the way openPort works it out, and has to be:
// a transcript is kept per directory, so asking about the repository
// instead of the task's worktree would answer with somebody else's
// sessions. A task with no worktree has had no session opened in one, and
// falls through to nothing rather than to the repository.
//
// The engine is looked up rather than named, because which file a session
// left behind is the engine's own answer: a name this build does not run is
// still a session somebody opened, and it is read back by nothing.
//
// since is when the terminal was handed over, and it is what keeps this
// from filing an afternoon twice: the directory holds every session ever
// opened in it, and only what was said after the window suspended itself
// belongs to the session it is coming back from.
func fileSessionPort(s *store.Store, r *board.Reader, engines map[string]engine.Engine) func(t view.Task, engineName string, since time.Time) (int, error) {
	return func(t view.Task, engineName string, since time.Time) (int, error) {
		eng, ok := engines[engineName]
		if s == nil || t.ID == "" || !ok {
			return 0, nil
		}

		turns, err := eng.Transcript(openDir(r, t, ""), since)
		if err != nil {
			return 0, err
		}

		// The sentence the session opened on is not a turn. openContext
		// wrote it about the task, the reader did not say it, and
		// openJournal has already recorded that the terminal was handed
		// over; filed as a turn it would read as the reader opening every
		// session by reciting the task's own description at it.
		opening := openContext(t)

		filed := 0

		for _, turn := range turns {
			if turn.Text == opening {
				continue
			}

			if err := task.Dialogue(s, subject(t), turn.By, turn.Text); err != nil {
				return filed, err
			}

			filed++
		}

		return filed, nil
	}
}
