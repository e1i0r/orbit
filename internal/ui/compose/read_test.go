package compose

// Reading the issue before writing the task.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/tracker"
)

// urlForm is the form with a Linear URL parsed into it.
func urlForm(t *testing.T, readable bool, iss tracker.Issue) (State, Env) {
	t.Helper()

	s, e := form(t)
	s.tab = composeTabURL
	s.parsedIssue = &iss
	s.readable = readable
	s.id.SetValue(iss.ID)
	s.text.SetValue(iss.Title)

	return s, e
}

// TestATaskThatIsOnlyALinkIsRefused. A URL and a slug are a name, not a
// task: orbit could not read the body, and a headless run cannot either —
// its tool calls are auto-denied with nobody there to approve them. What it
// does then is invent the requirements or nothing at all, and both cost a
// run.
func TestATaskThatIsOnlyALinkIsRefused(t *testing.T) {
	s, e := urlForm(t, false, tracker.Issue{
		Kind:   "linear",
		ID:     "FRA-71",
		Title:  "prueba de extremo a extremo",
		RawURL: "https://linear.app/frauddi/issue/FRA-71/prueba-de-extremo-a-extremo",
	})

	_, out := s.Submit(true, e)
	if out.Write != nil {
		t.Fatal("a task that is only a link was written down")
	}

	for _, want := range []string{"write what has to be done", "LINEAR_API_KEY"} {
		if !strings.Contains(out.Said, want) {
			t.Errorf("the refusal says %q, want it to mention %q", out.Said, want)
		}
	}
}

// TestSomethingWrittenByHandIsEnough: the guard is about a task with nothing
// in it, not about tasks that came from a tracker.
func TestSomethingWrittenByHandIsEnough(t *testing.T) {
	s, _ := urlForm(t, false, tracker.Issue{Kind: "linear", ID: "FRA-71", Title: "the slug"})
	s.text.SetValue("the slug\n\nadd ago() to internal/ui and test it")

	if s.onlyALink(s.text.String()) {
		t.Error("a task the reader wrote into is refused as a bare link")
	}
}

// TestABodyThatCanBeReadIsReadBeforeTheTaskIsWritten, and the save the
// reader asked for finishes once it lands.
func TestABodyThatCanBeReadIsReadBeforeTheTaskIsWritten(t *testing.T) {
	s, e := urlForm(t, true, tracker.Issue{Kind: "linear", ID: "FRA-71", Title: "the slug"})

	read := tracker.Issue{Kind: "linear", ID: "FRA-71", Title: "the real title", Description: "the real body"}
	e.Read = func(tracker.Issue) (tracker.Issue, error) { return read, nil }

	if !s.needsBody() {
		t.Fatal("a readable issue with no body does not ask for one")
	}

	next, out := s.Submit(true, e)
	if out.Cmd == nil || !out.Waiting {
		t.Fatal("nothing was asked of the tracker")
	}

	if _, waiting := next.Reading(); !waiting {
		t.Error("the form does not say it is waiting on the tracker")
	}

	// The sentence in the band says what is out, like everything else that
	// is waiting.
	wantBand(t, out, "reading")

	// What comes back goes into the form, and finishes the save.
	msg, ok := out.Cmd().(ReadMsg)
	if !ok {
		t.Fatalf("the tracker answered with %T, want the form's own message", out.Cmd())
	}

	after, out := next.Took(msg, e)
	if _, waiting := after.Reading(); waiting {
		t.Error("the form is still waiting after the answer landed")
	}

	if out.Write == nil || !strings.Contains(out.Write.Text, "the real body") {
		t.Errorf("the task written is %+v, want the body that came back", out.Write)
	}
}

// TestAnIssueThatCouldNotBeReadFallsBackToAskingTheReader.
func TestAnIssueThatCouldNotBeReadFallsBackToAskingTheReader(t *testing.T) {
	s, e := urlForm(t, true, tracker.Issue{Kind: "linear", ID: "FRA-71", Title: "the slug"})
	s.reading = true

	after, out := s.Took(ReadMsg{id: "FRA-71", err: tracker.ErrNoKey}, e)
	if after.readable {
		t.Error("a tracker that refused is still believed readable")
	}

	wantBand(t, out, "could not read")
}

// TestAnIssueWithNoBodyIsAskedForOnce. A tracker that answers with an empty
// description is answering successfully, and the form used to read that as
// "no body yet": it asked again, and again, one call to the tracker for
// every turn, for as long as the form stayed open.
func TestAnIssueWithNoBodyIsAskedForOnce(t *testing.T) {
	s, e := urlForm(t, true, tracker.Issue{Kind: "linear", ID: "FRA-71", Title: "the slug"})
	s.reading = true

	after, out := s.Took(ReadMsg{id: "FRA-71", issue: tracker.Issue{
		Kind: "linear", ID: "FRA-71", Title: "the slug",
	}}, e)

	if after.needsBody() {
		t.Error("the form would ask the tracker about that issue again")
	}

	wantBand(t, out, "no description")
}

// TestAnAnswerAboutAnotherIssueIsDropped. The reader pressed esc, or opened
// the form again for something else, while the tracker was still answering:
// what comes back belongs to a question they have left behind.
func TestAnAnswerAboutAnotherIssueIsDropped(t *testing.T) {
	s, e := urlForm(t, true, tracker.Issue{Kind: "linear", ID: "FRA-71", Title: "the slug"})
	s.reading = true

	after, _ := s.Took(ReadMsg{id: "FRA-70", issue: tracker.Issue{
		Kind: "linear", ID: "FRA-70", Title: "another", Description: "another body",
	}}, e)

	if strings.Contains(after.text.String(), "another body") {
		t.Errorf("an answer about another issue landed in the form: %q", after.text.String())
	}

	if _, waiting := after.Reading(); !waiting {
		t.Error("an answer about another issue stopped this one's wait")
	}
}

// TestASecondPressWhileReadingWritesNothing. Both guards are asleep in that
// window — the machine says it can read the issue, and a read is already out
// — so the second press used to write the task with its title as the whole
// body, which is what those guards exist to stop.
func TestASecondPressWhileReadingWritesNothing(t *testing.T) {
	s, e := urlForm(t, true, tracker.Issue{Kind: "linear", ID: "FRA-71", Title: "the slug"})
	s.reading = true

	_, out := s.Submit(false, e)
	if out.Write != nil {
		t.Errorf("a second press while the tracker was answering wrote %+v", out.Write)
	}

	wantBand(t, out, "still reading")
}
