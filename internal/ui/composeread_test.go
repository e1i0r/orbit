package ui

// Reading the issue before writing the task.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/tracker"
)

// urlForm is the compose screen with a Linear URL parsed into it.
func urlForm(t *testing.T, readable bool, iss tracker.Issue) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)
	m = m.openCompose()
	m.compose.tab = composeTabURL
	m.compose.parsedIssue = &iss
	m.compose.readable = readable
	m.compose.id.SetValue(iss.ID)
	m.compose.text.SetValue(iss.Title)

	return m
}

// TestATaskThatIsOnlyALinkIsRefused. A URL and a slug are a name, not a
// task: orbit could not read the body, and a headless run cannot either —
// its tool calls are auto-denied with nobody there to approve them. What it
// does then is invent the requirements or nothing at all, and both cost a
// run.
func TestATaskThatIsOnlyALinkIsRefused(t *testing.T) {
	m := urlForm(t, false, tracker.Issue{
		Kind:   "linear",
		ID:     "FRA-71",
		Title:  "prueba de extremo a extremo",
		RawURL: "https://linear.app/frauddi/issue/FRA-71/prueba-de-extremo-a-extremo",
	})

	next, cmd := m.composeSubmit(true)
	if cmd != nil {
		t.Fatal("a task that is only a link was written down")
	}

	said := asModel(t, next).message
	for _, want := range []string{"write what has to be done", "LINEAR_API_KEY"} {
		if !strings.Contains(said, want) {
			t.Errorf("the refusal says %q, want it to mention %q", said, want)
		}
	}
}

// TestSomethingWrittenByHandIsEnough: the guard is about a task with nothing
// in it, not about tasks that came from a tracker.
func TestSomethingWrittenByHandIsEnough(t *testing.T) {
	m := urlForm(t, false, tracker.Issue{Kind: "linear", ID: "FRA-71", Title: "the slug"})
	m.compose.text.SetValue("the slug\n\nadd ago() to internal/ui and test it")

	if m.onlyALink(m.compose.text.String()) {
		t.Error("a task the reader wrote into is refused as a bare link")
	}
}

// TestABodyThatCanBeReadIsReadBeforeTheTaskIsWritten, and the save the
// reader asked for finishes once it lands.
func TestABodyThatCanBeReadIsReadBeforeTheTaskIsWritten(t *testing.T) {
	m := urlForm(t, true, tracker.Issue{Kind: "linear", ID: "FRA-71", Title: "the slug"})

	if !m.needsBody() {
		t.Fatal("a readable issue with no body does not ask for one")
	}

	next, cmd := m.readIssue()
	if cmd == nil || !next.compose.reading {
		t.Fatal("nothing was asked of the tracker")
	}

	// The band says it while it waits, like everything else that is out.
	next.compose.startAfterRead = true
	if got := next.waitingLine(); !strings.Contains(got, "issue") {
		t.Errorf("the band says %q while the issue is being read", got)
	}

	// What comes back goes into the form.
	read := tracker.Issue{Kind: "linear", ID: "FRA-71", Title: "the real title", Description: "the real body"}

	done, _ := next.tookIssue(issueReadMsg{issue: read})

	after := asModel(t, done)
	if after.compose.reading {
		t.Error("the form is still waiting after the answer landed")
	}

	if !strings.Contains(after.compose.text.String(), "the real body") {
		t.Errorf("the task reads %q", after.compose.text.String())
	}
}

// TestAnIssueThatCouldNotBeReadFallsBackToAskingTheReader.
func TestAnIssueThatCouldNotBeReadFallsBackToAskingTheReader(t *testing.T) {
	m := urlForm(t, true, tracker.Issue{Kind: "linear", ID: "FRA-71", Title: "the slug"})
	m.compose.reading = true

	next, _ := m.tookIssue(issueReadMsg{err: tracker.ErrNoKey})

	after := asModel(t, next)
	if after.compose.readable {
		t.Error("a tracker that refused is still believed readable")
	}

	if !strings.Contains(after.message, "could not read") {
		t.Errorf("the band says %q", after.message)
	}
}
