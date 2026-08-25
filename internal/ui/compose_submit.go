package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/tracker"
)

// composeSubmit validates task fields and runs the new task creation command.
func (m Model) composeSubmit(startNow bool) (tea.Model, tea.Cmd) {
	p := m.opts.Words
	repo := strings.TrimSpace(m.compose.repo)
	if repo == "" && len(m.compose.repos) > 0 {
		repo = m.compose.repos[m.compose.repoIdx].name
	}
	id := strings.TrimSpace(m.compose.id)
	text := strings.TrimSpace(m.compose.text)

	if m.compose.tab == composeTabURL {
		if m.compose.parsedIssue != nil {
			if id == "" {
				id = m.compose.parsedIssue.ID
			}
			if text == "" {
				text = tracker.FormatPrompt(*m.compose.parsedIssue)
			}
		} else if strings.TrimSpace(m.compose.url) != "" {
			if issue, err := tracker.Parse(m.compose.url); err == nil {
				if id == "" {
					id = issue.ID
				}
				if text == "" {
					text = tracker.FormatPrompt(issue)
				}
			}
		}
	}

	switch {
	case repo == "":
		return m.say(p.T("compose.repo_required",
			"the repository is required; which one is this task against?")), nil
	case id == "":
		return m.say(p.T("compose.id_required",
			"the id is required; what is this task called?")), nil
	case text == "":
		return m.say(p.T("compose.text_required",
			"the task needs something written in it")), nil
	}
	if m.opts.ValidID != nil {
		if err := m.opts.ValidID(id); err != nil {
			return m.say(err.Error()), nil
		}
	}

	path := repo
	for _, r := range m.compose.repos {
		if r.name == repo {
			path = r.path
			break
		}
	}
	if path == repo {
		for _, t := range m.board.Tasks {
			if t.Repo == repo {
				path = t.RepoPath
				break
			}
		}
	}

	m.screen = screenList
	m.pendingID, m.pendTries = id, 0
	return m.runWatched(Command{Name: "new"}, []string{"-repo", path, "-id", id, "--", text})
}

// selectPending waits for newly created task to appear on the board.
func (m Model) selectPending() Model {
	if m.pendingID == "" {
		return m
	}
	for i, r := range m.rows() {
		if !r.head && !r.blank && r.task.ID == m.pendingID {
			m.pendingID, m.pendTries = "", 0
			return m.moveTo(i)
		}
	}
	m.pendTries++
	if m.pendTries > 2 {
		m.pendingID, m.pendTries = "", 0
	}
	return m
}
