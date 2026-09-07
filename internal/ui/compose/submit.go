package compose

import (
	"strings"

	"github.com/e1i0r/orbit/internal/tracker"
)

// Submit is the form answering with the task it holds, or with the sentence
// saying what is missing. It writes nothing itself: what a command does is
// the window's business.
func (s State) Submit(startNow bool, e Env) (State, Out) {
	p := e.Words

	path := strings.TrimSpace(s.repoPath)
	id := strings.TrimSpace(s.id.String())
	text := strings.TrimSpace(s.text.String())

	if s.tab == composeTabURL {
		if s.parsedIssue != nil {
			if id == "" {
				id = s.parsedIssue.ID
			}

			if text == "" {
				text = tracker.FormatPrompt(*s.parsedIssue)
			}
		} else if strings.TrimSpace(s.url.String()) != "" {
			if issue, err := tracker.Parse(s.url.String()); err == nil {
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
		return s, said(p.T("compose.id_required",
			"the id is required; what is this task called?"))
	case text == "":
		return s, said(p.T("compose.text_required",
			"the task needs something written in it"))
	case s.onlyALink(text):
		// A URL and a title are a name, not a task. Orbit cannot read the
		// body of this issue on this machine, and a headless run cannot
		// either: its tool calls are auto-denied with nobody there to
		// approve them. What it does then is invent the requirements or
		// nothing at all, and both cost a run.
		return s, said(p.T("compose.body_unreadable",
			"orbit cannot read this issue here, and a run will not be able to either: write what has to be done, or set LINEAR_API_KEY"))
	}

	// The body first, when this machine can read it: a task written from a
	// URL alone is a task nobody can follow. What comes back finishes this
	// same save — see read.go.
	if s.needsBody() {
		s.startAfterRead = startNow

		return s.readIssue(e)
	}

	if e.ValidID != nil {
		if err := e.ValidID(id); err != nil {
			return s, said(err.Error())
		}
	}

	// The knobs are left as the seat set them. The form no longer answers
	// which engine runs this, so it no longer overwrites the answer.
	//
	// The repository travels only when there is one. A board with no
	// repository on it writes a task against none, which runs in a
	// directory of its own and joins its first checkout when the work
	// reaches one; handing over an empty path would instead give it
	// whatever directory the window was started from.
	//
	// Start is what makes "save and start" true. It used to be taken and
	// dropped: the task was written down, the window went back to the
	// board, and the row sat in to do with nobody able to say why.
	return State{}, Out{
		Leave: true,
		Write: &Task{ID: id, Repo: path, Flow: s.chosenFlow(), Text: text, Start: startNow},
	}
}

// onlyALink is whether all this task would carry is what the URL itself
// says: the id, the slug, and a note telling somebody to go and look.
func (s State) onlyALink(text string) bool {
	iss := s.parsedIssue
	if iss == nil || s.readable || iss.Description != "" {
		return false
	}

	written := strings.TrimSpace(text)

	return written == "" || written == strings.TrimSpace(iss.Title)
}
