package ui

// Saying what a gesture is doing, and saying when it landed.
//
// Every verb on a task used to answer once, at the moment the key was
// pressed, in the past tense: "asked ORB-121 to cancel". Then nothing.
// The run stopped, or did not, a phase later, and the band had long since
// moved on to something else. Elio, having pressed x and watched a task go
// on spending for two minutes: I need to do it quickly, and with feedback,
// like cancelling... cancelled.
//
// So a gesture says two things now. The first is said the moment the key
// is pressed and is in the present: something is happening and Orbit is
// the one doing it. The second is said when the board comes back showing
// that it happened, which is the only moment anybody can honestly claim
// it did.
//
// The board is already polled twice a second, so the second sentence
// costs nothing but the remembering.

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// The gestures a task can be asked for, by the word the window uses for
// them. The control words are the store's own; the rest are named here
// because nothing else needed to name them.
const (
	gesturePause    = "pause"
	gestureResume   = "resume"
	gestureContinue = "continue"
	gestureCancel   = "cancel"
	gestureSkip     = "skip"
	gestureRequeue  = "requeue"
	gestureRead     = "read"
	gestureStart    = "start"
	gestureDelete   = "delete"
)

// awaitedFor is how long a gesture is waited on before the window stops
// promising it is coming.
//
// A cancel reaches a process in milliseconds and a pause waits for the
// phase to end, which is minutes. What this is guarding is neither: it is
// the gesture that will never land because the run died, or the marker was
// stale, or the word was written for a process that was already gone. The
// window stops claiming, and the board underneath is the truth either way.
const awaitedFor = 90 * time.Second

// awaited is the gesture this window is waiting to see land.
//
// One at a time. A reader presses one key and looks at the band, and two
// sentences racing for the same line is a band that says whichever won.
type awaited struct {
	id   string
	verb string
	at   time.Time
}

// awaiting remembers a gesture and says what to put on the band now.
func (m Model) awaiting(id, verb string) Model {
	m.await = awaited{id: id, verb: verb, at: m.now}

	return m.say(m.doingSaid(id, verb))
}

// doingSaid is the present tense, said when the key is pressed.
func (m Model) doingSaid(id, verb string) string {
	p, id_ := m.opts.Words, about("id", id)

	switch verb {
	case gesturePause:
		return p.T("doing.pause", "pausing {id}…", id_)
	case gestureResume:
		return p.T("doing.resume", "resuming {id}…", id_)
	case gestureContinue:
		return p.T("doing.continue", "letting {id} carry on…", id_)
	case gestureCancel:
		return p.T("doing.cancel", "cancelling {id}…", id_)
	case gestureSkip:
		return p.T("doing.skip", "skipping this phase of {id}…", id_)
	case gestureRequeue:
		return p.T("doing.requeue", "putting {id} back in to do…", id_)
	case gestureRead:
		return p.T("doing.read", "marking {id} read…", id_)
	case gestureStart:
		return p.T("doing.start", "starting {id}…", id_)
	case gestureDelete:
		return p.T("doing.delete", "deleting {id}…", id_)
	}

	return p.T("doing.other", "{verb} on {id}…", id_, about("verb", verb))
}

// doneSaid is the past tense, said when the board shows it happened.
func (m Model) doneSaid(id, verb string) string {
	p, id_ := m.opts.Words, about("id", id)

	switch verb {
	case gesturePause:
		return p.T("landed.pause", "{id} is paused", id_)
	case gestureResume:
		return p.T("landed.resume", "{id} is running again", id_)
	case gestureContinue:
		return p.T("landed.continue", "{id} carried on", id_)
	case gestureCancel:
		return p.T("landed.cancel", "{id} cancelled", id_)
	case gestureSkip:
		return p.T("landed.skip", "{id} skipped that phase", id_)
	case gestureRequeue:
		return p.T("landed.requeue", "{id} is back in to do", id_)
	case gestureRead:
		return p.T("landed.read", "{id} is marked read", id_)
	case gestureStart:
		return p.T("landed.start", "{id} is running", id_)
	case gestureDelete:
		return p.T("landed.delete", "{id} is gone, with its worktree and its branch", id_)
	}

	return p.T("landed.other", "{verb} landed on {id}", id_, about("verb", verb))
}

// took answers the awaited gesture against the board that just arrived: the
// sentence to say, and whether the waiting is over.
//
// Three endings. It landed, and the past tense is said. It has been long
// enough that the window stops claiming. Or it is still out, and nothing is
// said at all — the band is holding the present tense and that is still
// true.
func (m Model) tookAwaited(b boardOf) (string, bool) {
	if m.await.id == "" {
		return "", false
	}

	t, on := b.task(m.await.id)
	if landed(m.await.verb, t, on) {
		return m.doneSaid(m.await.id, m.await.verb), true
	}

	if m.now.Sub(m.await.at) > awaitedFor {
		return "", true
	}

	return "", false
}

// boardOf is the one question took asks of a board, declared here so the
// answer can be given by a test without one.
type boardOf interface {
	task(id string) (view.Task, bool)
}

// landed is whether the board now shows what the gesture asked for.
//
// It reads the row and not the record, because the row is what the reader
// is looking at: a sentence that says a task is paused beside a row that
// says it is running is worse than no sentence.
//
// A gesture whose word is not here never lands, and the window stops
// claiming after awaitedFor rather than promising for ever.
func landed(verb string, t view.Task, on bool) bool {
	if verb == gestureRead {
		return on && t.Read
	}

	// A task that has left the board did what was asked of it, whatever
	// was asked: there is nothing else it could have done.
	if !on {
		return true
	}

	switch verb {
	case gestureCancel:
		// Not running any more, whichever way it stopped. A cancel that
		// raced a run reaching its own end is still a task that stopped.
		return view.BandOf(t) != view.Running
	case gesturePause:
		return view.BandOf(t) == view.NeedsYou || t.Live == view.LiveFree
	case gestureResume, gestureContinue, gestureSkip:
		return view.BandOf(t) == view.Running
	case gestureRequeue:
		return view.BandOf(t) == view.ToDo
	case gestureStart:
		return view.BandOf(t) == view.Running
	case gestureDelete:
		// Only the row leaving the board, which the clause above answers.
		// A task still listed is a task still there, whatever else the
		// delete managed to remove.
		return false
	}

	return false
}

// awaitedWord is what one task's status cell says while a gesture on it is
// still out, and whether there is one.
//
// The word is short because the cell is: "cancelling…", not "cancelling
// ORB-121…". The row is already the task.
func (m Model) awaitedWord(id string) (string, bool) {
	if m.await.id != id || m.await.id == "" {
		return "", false
	}

	p := m.opts.Words

	switch m.await.verb {
	case gesturePause:
		return p.T("cell.pausing", "pausing…"), true
	case gestureResume:
		return p.T("cell.resuming", "resuming…"), true
	case gestureContinue:
		return p.T("cell.continuing", "carrying on…"), true
	case gestureCancel:
		return p.T("cell.cancelling", "cancelling…"), true
	case gestureSkip:
		return p.T("cell.skipping", "skipping…"), true
	case gestureRequeue:
		return p.T("cell.requeuing", "putting back…"), true
	case gestureRead:
		return p.T("cell.reading", "marking read…"), true
	case gestureStart:
		return p.T("cell.starting", "starting…"), true
	case gestureDelete:
		return p.T("cell.deleting", "deleting…"), true
	}

	return "", false
}

// stop signals the run holding a task, off the draw loop like every other
// port call.
//
// It is here rather than with the other port commands because it is half
// of this file's subject: the quick half of a cancel. It answers through
// controlMsg, which is what the window already reads a stop's failure out
// of, and a nil port is the window falling back to the control word.
func stop(port func(view.Task) error, t view.Task) tea.Cmd {
	return func() tea.Msg {
		return controlMsg{ID: t.ID, Word: gestureCancel, Err: port(t)}
	}
}

// deleted removes a task and everything of it, off the draw loop.
//
// It used to run inside the keypress: the port removes a worktree and a
// branch with git, and the window sat frozen through it and then said
// "task ORB-120 deleted" to a reader who had been looking at a still
// picture. Now the row says "deleting…" while it happens, like every
// other gesture.
func deleted(port func(view.Task) error, rescan func() error, t view.Task) tea.Cmd {
	return func() tea.Msg {
		if port == nil {
			return deletedMsg{ID: t.ID}
		}

		if err := port(t); err != nil {
			return deletedMsg{ID: t.ID, Err: err}
		}

		// The board is re-read here rather than waited for: a delete
		// changes which repositories have tasks in them, which the poll
		// alone does not notice.
		if rescan != nil {
			if err := rescan(); err != nil {
				return deletedMsg{ID: t.ID, Err: err}
			}
		}

		return deletedMsg{ID: t.ID}
	}
}

// deletedMsg is a delete coming back.
type deletedMsg struct {
	ID  string
	Err error
}
