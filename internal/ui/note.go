package ui

// The message box: one line of typing, handed to a command about a task.
//
// Two commands take a message and are otherwise nothing alike. note leaves
// text for the next phase to read, and direct interrupts the run to say it
// now — recording the directive, then stopping the process so the next start
// reads it. They share the box because typing a sentence is the same act;
// what is done with the sentence is the command's business.

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/clip"

	"github.com/e1i0r/orbit/internal/view"
)

// verbNote and verbDirect are the two children the box can be opened for,
// both of the task family. The box is handed the parent to run and the
// child to run it with.
const (
	verbNote   = "note"
	verbDirect = "direct"
)

// noteState is the box while it is up: which command the typing goes to,
// which of its children, the task it is about, and what has been typed so
// far.
type noteState struct {
	open   bool
	verb   string
	child  string
	taskID string
	text   string
}

// taskInHand is the task the window is about right now: the one being read,
// or the row the cursor is on, and nothing when the cursor is on a heading or
// there is no board to have a cursor in.
func (m Model) taskInHand() string {
	if m.detail != "" {
		return m.detail
	}

	if r, ok := m.selected(); ok && !r.head {
		return r.task.ID
	}

	return ""
}

// openMessage brings the box up for one command about one task.
func (m Model) openMessage(verb, child, taskID string) Model {
	if taskID == "" {
		return m
	}

	m.note = noteState{open: true, verb: verb, child: child, taskID: taskID}

	return m
}

// openNote is the box on the note key, about whatever task is in hand.
func (m Model) openNote() Model {
	return m.openMessage("task", verbNote, m.taskInHand())
}

func (m Model) closeNote() Model {
	m.note = noteState{}
	return m
}

func (m Model) noteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape || key.Matches(msg, m.keys.Back):
		return m.closeNote(), nil
	case msg.Code == tea.KeyEnter || key.Matches(msg, m.keys.Open):
		return m.submitNote()
	case (msg.Code == 'v' || msg.Code == 'V') && msg.Mod&tea.ModCtrl != 0:
		if clip := clip.Read(); clip != "" {
			m.note.text += clip
		}

		return m, nil
	case msg.Code == tea.KeyBackspace || msg.Code == tea.KeyDelete:
		m.note.text = cells.TrimLastRune(m.note.text)
		return m, nil
	}

	if msg.Text != "" {
		m.note.text += msg.Text
	}

	return m, nil
}

// submitNote hands what was typed to the command the box was opened for.
//
// The repository comes from the same place every other verb about a task
// takes it: the row on the board, and nothing at all when the task was
// written against no repository.
func (m Model) submitNote() (tea.Model, tea.Cmd) {
	p := m.opts.Words

	text := strings.TrimSpace(m.note.text)
	if text == "" {
		if m.note.child == verbDirect {
			return m.say(p.T("direct.empty", "the directive cannot be empty")), nil
		}

		return m.say(p.T("note.empty", "the note cannot be empty")), nil
	}

	verb, child, taskID := m.note.verb, m.note.child, m.note.taskID
	m.note = noteState{}

	said := p.T("note.recorded", "note recorded for {id}", about("id", taskID))
	// On a task that is running, the note is not filed and forgotten: the
	// next phase to start reads it (task.unconsumedNotes), which is the
	// whole reason a reader leaves one mid-run.
	if view.BandOf(m.subject()) == view.Running {
		said = p.T("note.recorded_running", "note recorded for {id} — the next phase reads it",
			about("id", taskID))
	}

	if child == verbDirect {
		said = p.T("direct.given", "{id} redirected — the run it was in is stopped",
			about("id", taskID))
	}

	m = m.say(said)

	var cmds []tea.Cmd
	if m.opts.Reader != nil && m.screen == screenDetail {
		cmds = append(cmds, logOf(m.opts.Reader, m.subject()))
	}

	nextM, runCmd := m.runWatchedSaying(Command{Name: verb},
		append([]string{child}, repoArgs(m.taskRepoPath(taskID), taskID, text)...), said)
	if runCmd != nil {
		cmds = append(cmds, runCmd)
	}

	return nextM, tea.Batch(cmds...)
}
