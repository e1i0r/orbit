package tracker

import (
	"testing"
)

// TestAForeignHostIsNotATracker is the fix.
//
// Every Match asked strings.Contains over the whole URL, so a link whose
// path merely mentioned a tracker was recognised as that tracker: it was
// accepted on the paste, parsed, and the prompt handed to an engine said
// Linear while carrying a link somewhere else entirely. The host is the only
// part of a URL that says who answers it.
func TestAForeignHostIsNotATracker(t *testing.T) {
	for _, rawURL := range []string{
		"https://evil.example/linear.app/issue/ENG-1/fix-it",
		"https://evil.example/?next=https://linear.app/acme/issue/ENG-1",
		"https://linear.app.evil.example/acme/issue/ENG-1",
		"https://evil.example/jira.atlassian.net/browse/PROJ-1",
		"https://evil.example/github.com/acme/api/issues/42",
		"https://evil.example/gitlab.com/acme/api/-/issues/42",
		"https://notjira.example/browse/PROJ-1",
	} {
		if IsTrackerURL(rawURL) {
			t.Errorf("%q was taken for a tracker issue", rawURL)
		}

		if iss, err := Parse(rawURL); err == nil {
			t.Errorf("%q parsed as a %s issue %s", rawURL, iss.Kind, iss.ID)
		}
	}
}

// TestTheRealTrackersStillMatch. Tightening the rule must not lose the URLs
// a reader actually pastes, subdomains and www included.
func TestTheRealTrackersStillMatch(t *testing.T) {
	for _, c := range []struct{ url, kind, id string }{
		{"https://linear.app/acme/issue/ENG-123/fix-the-thing", "linear", "ENG-123"},
		{"linear.app/acme/issue/ENG-123", "linear", "ENG-123"},
		{"https://acme.atlassian.net/browse/PROJ-45", "jira", "PROJ-45"},
		{"https://jira.acme.example/browse/PROJ-45", "jira", "PROJ-45"},
		{"https://github.com/acme/api/issues/42", "github", "GH-42"},
		{"https://www.github.com/acme/api/issues/42", "github", "GH-42"},
		{"https://gitlab.com/acme/api/-/issues/42", "gitlab", "GL-42"},
	} {
		if !IsTrackerURL(c.url) {
			t.Errorf("%q was not recognised as a tracker issue", c.url)
		}

		iss, err := Parse(c.url)
		if err != nil {
			t.Errorf("Parse(%q): %v", c.url, err)
			continue
		}

		if iss.Kind != c.kind || iss.ID != c.id {
			t.Errorf("Parse(%q) = %s %s, want %s %s", c.url, iss.Kind, iss.ID, c.kind, c.id)
		}
	}
}
