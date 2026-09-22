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

	// A second press while the tracker is answering is not a second task.
	// Both guards below are asleep in that window — onlyALink because the
	// machine says it can read the issue, and needsBody because a read is
	// already out — so without this the form would write the task with the
	// title as its whole body, which is the one thing those two exist to
	// stop.
	if s.reading {
		return refuse(s, p.T("compose.still_reading", "still reading the issue; give it a moment"))
	}

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
		return refuse(s, p.T("compose.id_required",
			"the id is required; what is this task called?"))
	case text == "":
		return refuse(s, p.T("compose.text_required",
			"the task needs something written in it"))
	case s.emptyIssue(text):
		// The tracker answered and the issue had no description in it. It
		// is a name, the same as a link is, and it is worth its own
		// sentence: nothing is wrong with the key or the connection, and a
		// reader told "orbit cannot read this issue" would go looking for
		// a fault that is not there.
		return refuse(s, p.T("compose.issue_empty",
			"that issue has no description; write what has to be done"))
	case s.onlyALink(text):
		// A URL and a title are a name, not a task. Orbit cannot read the
		// body of this issue on this machine, and a headless run cannot
		// either: its tool calls are auto-denied with nobody there to
		// approve them. What it does then is invent the requirements or
		// nothing at all, and both cost a run.
		return refuse(s, p.T("compose.body_unreadable",
			"orbit cannot read this issue here, and a run will not be able to either: write what has to be done, or set LINEAR_API_KEY"))
	}

	// The body first, when this machine can read it: a task written from a
	// URL alone is a task nobody can follow. What comes back finishes this
	// same save — see read.go.
	if s.needsBody() {
		s.startAfterRead = startNow

		return s.readIssue(e)
	}

	// The store's own id rule, phrased by the store. It is the refusal
	// the reader is most likely to earn and most able to fix, so it goes
	// on the form beside the box the id is typed in.
	if e.ValidID != nil {
		if err := e.ValidID(id); err != nil {
			return refuse(s, err.Error())
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
	// The form is handed back whole rather than emptied. The window is
	// leaving it — Leave puts the board up — but the write has not
	// happened yet, and a write that fails comes back to this form with
	// what was typed still in it. Emptying here threw that away before
	// anybody knew whether it was needed, so a refused save cost the
	// reader the task as well as the save. The window clears it when the
	// write lands.
	return s, Out{
		Leave: true,
		Write: &Task{ID: id, Repo: path, Flow: s.chosenFlow(), Text: text, Start: startNow},
	}
}

// emptyIssue is whether the tracker answered about this issue and there was
// nothing written in it — so all the task would carry is its own title.
func (s State) emptyIssue(text string) bool {
	iss := s.parsedIssue
	if iss == nil || !s.asked || iss.Description != "" {
		return false
	}

	written := strings.TrimSpace(text)

	return written == "" || written == strings.TrimSpace(iss.Title)
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

// refuse is the form saying why it will not write this task: on the form,
// beside the field that has to change, and on the band for the reader who
// was watching that instead.
//
// Both, because the two are read at different moments. These used to be
// the band alone, which put "the id is required" at the foot of the window
// under a form whose id box is the thing it is about.
func refuse(s State, why string) (State, Out) {
	return s.Refused(why), said(why)
}

// Refused is the form told why the window could not write the task, with
// everything that was typed still in it.
//
// The form's refusals used to be a sentence for the band and nothing
// else. The band is a line at the foot of the window, under whatever
// screen is up, that holds what it is given for a few seconds — and a
// reader who has just pressed Save is looking at the form. A save that
// failed and a save that worked were the same screen for as long as it
// took somebody to notice the task was not on the board.
//
// A door rather than a field the window writes, because the state is this
// package's: the window hands over the sentence the command answered with
// and this decides where it is drawn. The next keystroke clears it, since
// a refusal that outlived the keystroke answering it would be the form
// arguing with a field that has already changed.
func (s State) Refused(why string) State {
	s.refused = why

	return s
}
