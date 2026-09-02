package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/tracker"
)

// composeSubmit validates task fields and runs the new task creation command.
func (m Model) composeSubmit(startNow bool) (tea.Model, tea.Cmd) {
	p := m.opts.Words

	path := strings.TrimSpace(m.compose.repoPath)
	id := strings.TrimSpace(m.compose.id.String())
	text := strings.TrimSpace(m.compose.text.String())

	if m.compose.tab == composeTabURL {
		if m.compose.parsedIssue != nil {
			if id == "" {
				id = m.compose.parsedIssue.ID
			}

			if text == "" {
				text = tracker.FormatPrompt(*m.compose.parsedIssue)
			}
		} else if strings.TrimSpace(m.compose.url.String()) != "" {
			if issue, err := tracker.Parse(m.compose.url.String()); err == nil {
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

	// The knobs are left as the seat set them. The form no longer answers
	// which engine runs this, so it no longer overwrites the answer.
	flowName := m.compose.chosenFlow()

	// -repo only when there is one. A board with no repository on it writes
	// a task against none, which runs in a directory of its own and joins
	// its first checkout when the work reaches one; passing an empty path
	// would instead hand it whatever directory the window was started from.
	args := []string{"-id", id}
	if path != "" {
		args = append(args, "-repo", path)
	}

	if flowName != "" {
		args = append(args, "-flow", flowName)
	}

	args = append(args, "--", text)

	m.screen = screenList
	m.pendingID, m.pendTries = id, 0

	return m.runWatched(Command{Name: "new"}, args)
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
