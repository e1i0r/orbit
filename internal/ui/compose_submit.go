package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/tracker"
)

// composeSubmit validates task fields and runs the new task creation command.
func (m Model) composeSubmit(startNow bool) (tea.Model, tea.Cmd) {
	p := m.opts.Words

	// A second press while the tracker is answering is not a second task.
	// Both guards below are asleep in that window — onlyALink because the
	// machine says it can read the issue, and needsBody because a read is
	// already out — so without this the form would write the task with the
	// title as its whole body, which is the one thing those two exist to
	// stop.
	if m.compose.reading {
		return m.say(p.T("compose.still_reading", "still reading the issue; give it a moment")), nil
	}

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
	case m.emptyIssue(text):
		// The tracker answered and the issue had no description in it. It
		// is a name, the same as a link is, and it is worth its own
		// sentence: nothing is wrong with the key or the connection, and a
		// reader told "orbit cannot read this issue" would go looking for
		// a fault that is not there.
		return m.say(p.T("compose.issue_empty",
			"that issue has no description; write what has to be done")), nil
	case m.onlyALink(text):
		// A URL and a title are a name, not a task. Orbit cannot read the
		// body of this issue on this machine, and a headless run cannot
		// either: its tool calls are auto-denied with nobody there to
		// approve them. What it does then is invent the requirements or
		// nothing at all, and both cost a run.
		return m.say(p.T("compose.body_unreadable",
			"orbit cannot read this issue here, and a run will not be able to either: write what has to be done, or set LINEAR_API_KEY")), nil
	}

	// The body first, when this machine can read it: a task written from a
	// URL alone is a task nobody can follow. What comes back finishes this
	// same save — see composeread.go.
	if m.needsBody() {
		m.compose.startAfterRead = startNow

		next, cmd := m.readIssue()

		return next, cmd
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

	// The button says "save and start" and this is what makes it true. It
	// used to take the flag and drop it: the task was written down, the
	// window went back to the board, and the row sat in to do with nobody
	// able to say why. Starting it here, in the same command that writes
	// it, also means nothing races the board's next refresh.
	if startNow {
		args = append(args, "-start")
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

// emptyIssue is whether the tracker answered about this issue and there was
// nothing written in it — so all the task would carry is its own title.
func (m Model) emptyIssue(text string) bool {
	iss := m.compose.parsedIssue
	if iss == nil || !m.compose.asked || iss.Description != "" {
		return false
	}

	written := strings.TrimSpace(text)

	return written == "" || written == strings.TrimSpace(iss.Title)
}

// onlyALink is whether all this task would carry is what the URL itself
// says: the id, the slug, and a note telling somebody to go and look.
func (m Model) onlyALink(text string) bool {
	iss := m.compose.parsedIssue
	if iss == nil || m.compose.readable || iss.Description != "" {
		return false
	}

	written := strings.TrimSpace(text)

	return written == "" || written == strings.TrimSpace(iss.Title)
}
