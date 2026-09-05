package ui

import (
	"errors"
	"os/exec"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

type cliEndedMsg struct {
	Engine string
	Repo   string
	// Task is the task the session was opened on, and it is the zero task
	// for a session opened on none. What the reader is asked when they come
	// back turns on it: a session on a task is work on that task, and a
	// window that offers to write a second task down for it is offering to
	// split the same work in two. It is the whole task and not the id
	// because reading the session back needs the worktree it ran in.
	Task view.Task
	// Started is when the terminal was handed over. The engine keeps one
	// transcript per directory, holding every session ever opened there, so
	// this is what tells the session that just ended from the ones before
	// it.
	Started time.Time
	Err     error
}

// sessionFiledMsg is what the reading back answered: how much of the
// conversation went into the task, or why none of it did.
type sessionFiledMsg struct {
	ID    string
	Turns int
	Err   error
}

func (m Model) launchInteractiveCLI() (Model, tea.Cmd) {
	eng := m.dialEngine(m.knobs.Engine)
	// The path and not the name. This read r.task.Repo — the column, which
	// is what the repository is called — and handed it to exec as a working
	// directory, so every session opened on a task started in a directory
	// that does not exist, or worse, in one that happened to.
	// The task the view is open on, and the row under the cursor only when
	// there is no view. A session opened from the task view read the board's
	// cursor, which is on whatever row the reader left it on: the sentence
	// the session was told named that task, and the directory it opened in
	// was that task's — or, on a cursor that had never moved onto a task,
	// the first repository on the board rather than this task's worktree.
	var t view.Task
	if pointed, ok := m.pointedAt(); ok {
		t = pointed
	}

	repoDir := t.RepoPath
	if repoDir == "" && len(m.board.RepoList) > 0 {
		repoDir = m.board.RepoList[0].Path
	}

	started := m.now

	cmd, err := m.openSession(t, eng, repoDir)
	if err != nil {
		return m.say(m.opts.Words.T("msg.cli_exec_error", "error running {engine}: {err}", about("engine", eng), about("err", err.Error()))), nil
	}

	return m.say(m.opts.Words.T("msg.opening_cli", "opening interactive session with {engine}...", about("engine", eng))), tea.ExecProcess(cmd, func(err error) tea.Msg {
		return cliEndedMsg{Engine: eng, Repo: repoDir, Task: t, Started: started, Err: err}
	})
}

// openSession is the command line the terminal is handed: the port's, with
// Orbit's own server configured in it, or a bare engine for a window that
// was given no port.
func (m Model) openSession(t view.Task, engineName, dir string) (*exec.Cmd, error) {
	if m.opts.Open != nil {
		cmd, err := m.opts.Open(t, engineName, dir)
		if err != nil {
			return nil, err
		}

		if cmd != nil {
			return cmd, nil
		}
	}

	cmd := exec.Command(engineName)
	cmd.Dir = dir

	return cmd, nil
}

func (m Model) handleCLIEnded(msg cliEndedMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		var exitErr *exec.ExitError
		if !errors.As(msg.Err, &exitErr) {
			return m.say(m.opts.Words.T("msg.cli_exec_error", "error running {engine}: {err}", about("engine", msg.Engine), about("err", m.errSaid(msg.Err)))), nil
		}
	}

	// A session opened on a task is already about one, so there is nothing
	// to ask: the question exists for the reader who opened a bare terminal
	// on a repository, worked something out in it, and has a task to write
	// down at the end of it.
	if msg.Task.ID != "" {
		return m.say(m.opts.Words.T("msg.cli_ended_on_task",
			"the session on {id} ended", about("id", msg.Task.ID))), fileSession(m.opts.FileSession, msg)
	}

	m.confirm = confirmPostCliTask
	m.confirmID = msg.Repo

	return m.say(m.opts.Words.T("msg.confirm_post_cli", "make a task from this session? [y/n]")), nil
}

// fileSession reads the conversation back into the task, off the event
// loop: it is a file the size of an afternoon's work, and the window has
// just come back from being suspended for that afternoon.
func fileSession(port func(view.Task, string, time.Time) (int, error), msg cliEndedMsg) tea.Cmd {
	if port == nil {
		return nil
	}

	return func() tea.Msg {
		turns, err := port(msg.Task, msg.Engine, msg.Started)

		return sessionFiledMsg{ID: msg.Task.ID, Turns: turns, Err: err}
	}
}

// handleSessionFiled says how much of the session came back, and asks for
// the record again when the reader is standing on the task it went into —
// the notes tab is drawn from entries this window is already holding, so
// without that the conversation is in the record and not on the screen.
func (m Model) handleSessionFiled(msg sessionFiledMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		return m.say(m.opts.Words.T("msg.session_not_filed",
			"the session on {id} could not be read back: {err}",
			about("id", msg.ID), about("err", m.errSaid(msg.Err)))), nil
	}

	// Nothing said is nothing to report. A reader who opened a terminal,
	// ran two commands in it and came out has had no conversation, and a
	// window announcing that is a window talking about its own plumbing.
	if msg.Turns == 0 {
		return m, nil
	}

	said := m.opts.Words.T("msg.session_filed", "{n} turns of that session are now in {id}",
		about("n", strconv.Itoa(msg.Turns)), about("id", msg.ID))
	if msg.Turns == 1 {
		said = m.opts.Words.T("msg.session_filed_one", "what was said in that session is now in {id}",
			about("id", msg.ID))
	}

	m = m.say(said)

	if m.screen != screenDetail || m.detail != msg.ID {
		return m, nil
	}

	return m, logOf(m.opts.Reader, m.subject())
}
