package ui

// Reading the issue before writing the task.
//
// A URL names an issue; the body is what says what to do. When this machine
// has a credential for that tracker, the form fetches it and the task is
// written with the real thing in it — and when it does not, compose_submit
// refuses to write a task whose whole content is a note telling somebody to
// go and look, because a headless run cannot follow it.

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/tracker"
)

// issueReadMsg is the issue, read.
type issueReadMsg struct {
	// id is the issue that was asked about. The answer is written in only
	// if the form is still on that issue: a reader who pressed esc, or who
	// reopened the form for something else, would otherwise have a reply to
	// a question they left behind land in the middle of what they are
	// typing now — the same guard every late answer in update.go carries.
	id    string
	issue tracker.Issue
	err   error
}

// needsBody is whether the form should fetch before it writes: an issue this
// machine can read, whose body it has not read yet.
func (m Model) needsBody() bool {
	iss := m.compose.parsedIssue

	return iss != nil && m.compose.readable && iss.Description == "" && !m.compose.reading && !m.compose.asked
}

// readIssue asks the tracker for the body, and says so while it waits.
func (m Model) readIssue() (Model, tea.Cmd) {
	iss := *m.compose.parsedIssue
	m.compose.reading = true
	m.compose.readAt = m.now

	m = m.say(m.opts.Words.T("compose.reading", "reading {id} from {kind}…",
		about("id", iss.ID), about("kind", iss.Kind)))

	next, frame := m.nextFrame()

	return next, tea.Batch(frame, func() tea.Msg {
		got, err := tracker.Read(context.Background(), iss)

		return issueReadMsg{id: iss.ID, issue: got, err: err}
	})
}

// tookIssue writes what came back into the form, and goes on with the save
// the reader already asked for.
//
// A body that could not be read is not a reason to stop: what it means is
// that the reader has to say what the task is, and compose_submit says so in
// those words. So the form comes back with what it had, and the same key
// they pressed will now be refused with a sentence they can act on.
func (m Model) tookIssue(msg issueReadMsg) (tea.Model, tea.Cmd) {
	if m.compose.parsedIssue == nil || m.compose.parsedIssue.ID != msg.id {
		return m, nil
	}

	m.compose.reading, m.compose.asked = false, true

	if msg.err != nil {
		m.compose.readable = false

		return m.say(m.opts.Words.T("compose.read_failed", "could not read the issue: {err}",
			about("err", m.errSaid(msg.err)))), nil
	}

	m.compose.parsedIssue = &msg.issue
	if msg.issue.Description != "" {
		m.compose.text.setValue(tracker.FormatPrompt(msg.issue))
	}

	return m.composeSubmit(m.compose.startAfterRead)
}
