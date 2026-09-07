package ui

// Where the window and the supervisor screen meet.
//
// The screen is handed the little world it was written against and answers
// with what it wants done. Everything the window knows and it does not —
// which screens there are, which engine is dialled, what the board has
// selected — is translated here and nowhere else.

import (
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/supervisor"
)

// supervisorEnv is the world the supervisor screen was written against.
func (m Model) supervisorEnv() supervisor.Env {
	e := supervisor.Env{
		Words:     m.opts.Words,
		Keys:      m.keys,
		Frame:     m.frame,
		Now:       m.now,
		Spinner:   m.spinner,
		Busy:      m.supervisorBusy,
		Knows:     m.opts.Knows,
		Record:    m.opts.RecordSupervisor,
		Retract:   m.opts.RetractSupervisor,
		Ask:       m.askSupervisor,
		NewID:     m.opts.NewConversation,
		Forget:    m.opts.RemoveConversation,
		Learn:     m.opts.Learn,
		Note:      m.opts.NoteTask,
		Engine:    m.dialEngine(m.knobs.Engine),
		IsEngine:  m.onTheRoster,
		Autopilot: m.autopilotOn(),
		Repo:      m.repoForFact(),
		Tasks:     m.mentions(),
	}

	// A window built without a reader has no thread to show, and the screen
	// is told that by being handed no door rather than one that answers
	// nothing.
	if m.opts.Reader != nil {
		e.Log = m.opts.Reader.SupervisorLog
	}

	return e
}

// askSupervisor is the question going out, as the command that carries the
// answer back into this window's own update loop.
func (m Model) askSupervisor(conversation, text string) tea.Cmd {
	return askSupervisorCmd(m.opts.AskSupervisor, m.dialEngine(m.knobs.Engine), conversation, text)
}

// onTheRoster is whether a name is one of the engines this build has.
func (m Model) onTheRoster(name string) bool {
	return slices.Contains(m.engineNames(), name)
}

// mentions is every task on the board, by id and title: an id alone is not a
// thing anybody remembers.
func (m Model) mentions() []supervisor.Mention {
	out := make([]supervisor.Mention, 0, len(m.board.Tasks))
	for _, t := range m.board.Tasks {
		out = append(out, supervisor.Mention{ID: t.ID, Title: t.Title})
	}

	return out
}

// repoForFact is the repository a fact with no scope is about: the one the
// task under the cursor is worked in, and otherwise the only one the board
// has. A board of several repositories with nothing selected cannot answer,
// and a fact that cannot say where it applies is written as a general one.
func (m Model) repoForFact() string {
	if r, ok := m.selected(); ok && r.task.RepoPath != "" {
		return r.task.RepoPath
	}

	if len(m.board.RepoList) == 1 {
		return m.board.RepoList[0].Path
	}

	return ""
}

// tookSupervisor does what the screen asked the window for.
func (m Model) tookSupervisor(next supervisor.State, out supervisor.Out) (Model, tea.Cmd) {
	m.supervisor = next

	if out.Leave {
		m.screen = screen(out.Back)
	}

	if out.Said != "" {
		m = m.say(out.Said)
	}

	cmd := out.Cmd

	if out.Asking {
		// The window owns the clock: the band says how long the question
		// has been out, and the spinner is turned by one chain of ticks.
		m.supervisorBusy, m.supervisorAt = true, m.now

		started, frame := m.nextFrame()
		m, cmd = started, tea.Batch(cmd, frame)
	}

	return m, cmd
}

// openSupervisor puts the screen up, on the conversation last spoken in.
func (m Model) openSupervisor() Model {
	prev := m.screen
	if prev == screenSupervisor {
		prev = screenList
	}

	m.supervisor = supervisor.Open(int(prev), m.supervisorEnv())
	m.screen = screenSupervisor

	return m
}

// supervisorKey hands one keystroke to the screen.
func (m Model) supervisorKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, out := m.supervisor.Key(msg, m.supervisorEnv())

	return m.tookSupervisor(next, out)
}

// sendSupervisorMessage puts one line in the thread and asks for an answer.
// It is how a caller with something of its own to say — the delivery keys,
// which send a line nobody typed — reaches the same door the keyboard does.
func (m Model) sendSupervisorMessage(text string) (Model, tea.Cmd) {
	next, out := m.supervisor.Say(text, m.supervisorEnv())

	said, cmd := m.tookSupervisor(next, out)
	if !out.Asking {
		// The thread refused the line. What it said about that is the only
		// true sentence there is here.
		return said, nil
	}

	return said, cmd
}

// syncSupervisor rereads the thread, for a window that has just changed what
// is in it.
func (m Model) syncSupervisor() Model {
	m.supervisor = m.supervisor.Sync(m.supervisorEnv())

	return m
}

// scrollThread moves the thread, for the wheel.
func (m Model) scrollThread(d int) Model {
	m.supervisor = m.supervisor.Wheel(d, m.supervisorEnv())

	return m
}

// supervisorRows is the screen drawn.
func (m Model) supervisorRows(h, w int) []string {
	return m.supervisor.View(h, w, m.supervisorEnv())
}
