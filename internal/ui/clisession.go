package ui

import (
	"errors"
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

type cliEndedMsg struct {
	Engine string
	Repo   string
	Err    error
}

func (m Model) launchInteractiveCLI() (Model, tea.Cmd) {
	eng := m.knobs.Engine
	if eng == "" {
		eng = "claude"
	}
	repoDir := ""
	if r, ok := m.selected(); ok && !r.head && r.task.Repo != "" {
		repoDir = r.task.Repo
	} else if len(m.board.RepoList) > 0 {
		repoDir = m.board.RepoList[0].Path
	}

	cmd := exec.Command(eng)
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	return m.say(m.opts.Words.T("msg.opening_cli", "opening interactive session with {engine}...", about("engine", eng))), tea.ExecProcess(cmd, func(err error) tea.Msg {
		return cliEndedMsg{Engine: eng, Repo: repoDir, Err: err}
	})
}

func (m Model) handleCLIEnded(msg cliEndedMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		var exitErr *exec.ExitError
		if !errors.As(msg.Err, &exitErr) {
			return m.say(m.opts.Words.T("msg.cli_exec_error", "error running {engine}: {err}", about("engine", msg.Engine), about("err", msg.Err.Error()))), nil
		}
	}
	m.confirm = confirmPostCliTask
	m.confirmID = msg.Repo
	return m.say(m.opts.Words.T("msg.confirm_post_cli", "create a task in Orbit from this session? press y to confirm, anything else to skip")), nil
}
