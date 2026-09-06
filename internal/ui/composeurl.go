package ui

// What the form does when the line it is on changes: a URL pasted anywhere
// in it is an issue, and the form follows it.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/tracker"
)

func (m *Model) onComposeChanged() {
	if m.compose.tab == composeTabURL && m.compose.field == composeURL {
		raw := strings.TrimSpace(m.compose.url.String())
		if raw != "" {
			if issue, err := tracker.Parse(raw); err == nil {
				m.compose.parsedIssue = &issue
				m.compose.readable = tracker.Readable(issue.Kind)

				m.compose.id.setValue(issue.ID)

				if issue.Title != "" {
					m.compose.text.setValue(issue.Title)
				}
			} else {
				m.compose.parsedIssue = nil
			}
		} else {
			m.compose.parsedIssue = nil
		}
	} else if m.compose.tab == composeTabManual {
		cur := strings.TrimSpace(m.compose.typed())
		if strings.HasPrefix(cur, "http://") || strings.HasPrefix(cur, "https://") ||
			strings.HasPrefix(cur, "linear.app/") {
			if issue, err := tracker.Parse(cur); err == nil {
				m.compose.tab = composeTabURL
				m.compose.field = composeURL
				m.compose.url.setValue(cur)
				m.compose.parsedIssue = &issue
				m.compose.readable = tracker.Readable(issue.Kind)

				m.compose.id.setValue(issue.ID)

				if issue.Title != "" {
					m.compose.text.setValue(issue.Title)
				}
			}
		}
	}
}

// active is the field being typed into, or nothing when the form is on a
// row of pills. Every key that writes, deletes or moves a caret goes
// through it, so which field a keystroke lands in is answered once.
