package compose

// What the form does when the line it is on changes: a URL pasted anywhere
// in it is an issue, and the form follows it.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/tracker"
)

func (s *State) onComposeChanged() {
	if s.tab == composeTabURL && s.field == composeURL {
		raw := strings.TrimSpace(s.url.String())
		if raw != "" {
			if issue, err := tracker.Parse(raw); err == nil {
				s.parsedIssue = &issue
				s.readable = tracker.Readable(issue.Kind)

				s.id.SetValue(issue.ID)

				if issue.Title != "" {
					s.text.SetValue(issue.Title)
				}
			} else {
				s.parsedIssue = nil
			}
		} else {
			s.parsedIssue = nil
		}
	} else if s.tab == composeTabManual {
		cur := strings.TrimSpace(s.typed())
		if strings.HasPrefix(cur, "http://") || strings.HasPrefix(cur, "https://") ||
			strings.HasPrefix(cur, "linear.app/") {
			if issue, err := tracker.Parse(cur); err == nil {
				s.tab = composeTabURL
				s.field = composeURL
				s.url.SetValue(cur)
				s.parsedIssue = &issue
				s.readable = tracker.Readable(issue.Kind)

				s.id.SetValue(issue.ID)

				if issue.Title != "" {
					s.text.SetValue(issue.Title)
				}
			}
		}
	}
}

// active is the field being typed into, or nothing when the form is on a
// row of pills. Every key that writes, deletes or moves a caret goes
// through it, so which field a keystroke lands in is answered once.
