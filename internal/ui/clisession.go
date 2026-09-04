package ui

import (
	"errors"
	"os/exec"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

type cliEndedMsg struct {
	Engine string
	Repo   string
	// TaskID is the task the session was opened on, and it is empty for a
	// session opened on none. What the reader is asked when they come back
	// turns on it: a session on a task is work on that task, and a window
	// that offers to write a second task down for it is offering to split
	// the same work in two.
	TaskID string
	Err    error
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

	cmd, err := m.openSession(t, eng, repoDir)
	if err != nil {
		return m.say(m.opts.Words.T("msg.cli_exec_error", "error running {engine}: {err}", about("engine", eng), about("err", err.Error()))), nil
	}

	return m.say(m.opts.Words.T("msg.opening_cli", "opening interactive session with {engine}...", about("engine", eng))), tea.ExecProcess(cmd, func(err error) tea.Msg {
		return cliEndedMsg{Engine: eng, Repo: repoDir, TaskID: t.ID, Err: err}
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
	if msg.TaskID != "" {
		return m.say(m.opts.Words.T("msg.cli_ended_on_task",
			"the session on {id} ended", about("id", msg.TaskID))), nil
	}

	m.confirm = confirmPostCliTask
	m.confirmID = msg.Repo

	return m.say(m.opts.Words.T("msg.confirm_post_cli", "create a task in Orbit from this session? press y to confirm, anything else to skip")), nil
}
