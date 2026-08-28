package tracker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

// TestFetchLinearReportsWhatTheAPIRefused is the fix.
//
// GraphQL answers 200 to a request it refused, with the reason in errors and
// data.issue null. Reading only data meant a revoked key and an issue on
// another workspace both returned ("", "", nil) — an empty issue and no
// error — and a task was created from it.
func TestFetchLinearReportsWhatTheAPIRefused(t *testing.T) {
	for _, c := range []struct {
		name, body, want string
	}{
		{
			"a revoked key",
			`{"errors":[{"message":"Authentication required, not authenticated"}],"data":null}`,
			"not authenticated",
		},
		{
			"an issue this key cannot see",
			`{"data":{"issue":null}}`,
			"ENG-1",
		},
	} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if _, err := w.Write([]byte(c.body)); err != nil {
				t.Errorf("write body: %v", err)
			}
		}))

		old := linearEndpoint
		linearEndpoint = ts.URL

		title, desc, err := FetchLinear(context.Background(), "lin_api_key", "ENG-1")

		linearEndpoint = old

		ts.Close()

		if err == nil {
			t.Errorf("%s: FetchLinear returned title %q desc %q and no error", c.name, title, desc)
			continue
		}

		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error = %v, want it to mention %q", c.name, err, c.want)
		}
	}
}

// TestFetchLinearReturnsTheIssue keeps the happy path honest: the guards
// above must not turn a real answer into a refusal.
func TestFetchLinearReturnsTheIssue(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(`{"data":{"issue":{"title":"Fix the thing","description":"It is broken"}}}`)); err != nil {
			t.Errorf("write body: %v", err)
		}
	}))
	defer ts.Close()

	old := linearEndpoint
	linearEndpoint = ts.URL

	defer func() { linearEndpoint = old }()

	title, desc, err := FetchLinear(context.Background(), "lin_api_key", "ENG-1")
	if err != nil {
		t.Fatalf("FetchLinear: %v", err)
	}

	if title != "Fix the thing" || desc != "It is broken" {
		t.Errorf("got %q / %q, want the issue's title and description", title, desc)
	}
}

// TestRegisterIsSafeUnderTheRaceDetector. Register appended to a
// package-level slice while every reader ranged over the same one, with
// nothing serialising them.
func TestRegisterIsSafeUnderTheRaceDetector(t *testing.T) {
	var wg sync.WaitGroup

	for range 8 {
		wg.Add(2)

		go func() {
			defer wg.Done()

			Register(LinearProvider{})
		}()

		go func() {
			defer wg.Done()

			IsTrackerURL("https://linear.app/acme/issue/ENG-1")
		}()
	}

	wg.Wait()
}
