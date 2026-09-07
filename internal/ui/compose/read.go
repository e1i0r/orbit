package compose

// Reading the issue before writing the task.
//
// A URL names an issue; the body is what says what to do. When this machine
// has a credential for that tracker, the form fetches it and the task is
// written with the real thing in it — and when it does not, compose_submit
// refuses to write a task whose whole content is a note telling somebody to
// go and look, because a headless run cannot follow it.

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/tracker"
)

// ReadMsg is the issue, read. It is this screen's own message: the window
// routes it back here and nothing else looks at it.
type ReadMsg struct {
	// id is the issue that was asked about. The answer is written in only
	// if the form is still on that issue: a reader who pressed esc, or who
	// reopened the form for something else, would otherwise have a reply to
	// a question they left behind land in the middle of what they are
	// typing now — the same guard every late answer the window routes has.
	id    string
	issue tracker.Issue
	err   error
}

// Reading is whether a body is being fetched right now, and since when. The
// window's band says how long a wait has been going on, and this is that
// wait.
func (s State) Reading() (since time.Time, waiting bool) {
	return s.readAt, s.reading
}

// needsBody is whether the form should fetch before it writes: an issue this
// machine can read, whose body it has not read yet.
func (s State) needsBody() bool {
	iss := s.parsedIssue

	return iss != nil && s.readable && iss.Description == "" && !s.reading && !s.asked
}

// readIssue asks the tracker for the body, and says so while it waits.
func (s State) readIssue(e Env) (State, Out) {
	iss := *s.parsedIssue
	s.reading = true
	s.readAt = e.Now

	read := e.Read
	if read == nil {
		read = func(iss tracker.Issue) (tracker.Issue, error) {
			return tracker.Read(context.Background(), iss)
		}
	}

	return s, Out{
		Said: e.Words.T("compose.reading", "reading {id} from {kind}…",
			about("id", iss.ID), about("kind", iss.Kind)),
		Waiting: true,
		Cmd: func() tea.Msg {
			got, err := read(iss)

			return ReadMsg{id: iss.ID, issue: got, err: err}
		},
	}
}

// Took writes what came back into the form, and goes on with the save
// the reader already asked for.
//
// A body that could not be read is not a reason to stop: what it means is
// that the reader has to say what the task is, and compose_submit says so in
// those words. So the form comes back with what it had, and the same key
// they pressed will now be refused with a sentence they can act on.
func (s State) Took(msg ReadMsg, e Env) (State, Out) {
	if s.parsedIssue == nil || s.parsedIssue.ID != msg.id {
		return s, Out{}
	}

	s.reading, s.asked = false, true

	if msg.err != nil {
		s.readable = false

		return s, said(e.Words.T("compose.read_failed", "could not read the issue: {err}",
			about("err", msg.err.Error())))
	}

	s.parsedIssue = &msg.issue
	if msg.issue.Description != "" {
		s.text.SetValue(tracker.FormatPrompt(msg.issue))
	}

	return s.Submit(s.startAfterRead, e)
}
