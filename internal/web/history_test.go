package web

// The conversation about one task, asked for over HTTP.
//
// It read half. The half that was missing is every case where there is
// nothing to read — a build with no thread behind it, and a thread that
// could not be read — and both of them are the page's own promise: nothing
// is hidden, so a screen that could not read something says what stopped it
// rather than drawing an empty conversation.

import (
	"errors"
	"net/http"
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
)

// aThread is a stand-in for whatever holds the conversation about a task.
type aThread struct {
	text string
	err  error
}

func (a aThread) History(string, string) (string, error) { return a.text, a.err }

// withThread is the server built with that thread behind it.
func withThread(tasks []view.Task, told Told) *Server {
	return New(Ports{
		Board: aBoard{tasks: tasks},
		Trees: nowhere{path: "/nowhere"},
		Root:  "/code",
		Files: built,
		Told:  told,
	})
}

// TestTheConversationAboutATaskComesBackWithIt.
func TestTheConversationAboutATaskComesBackWithIt(t *testing.T) {
	s := withThread(
		[]view.Task{{ID: "LED-1", Title: "fix the total", Band: view.ToDo, Repo: "ledger"}},
		aThread{text: "you: start it\norbit: started"},
	)

	code, body := ask(t, s, "/api/tasks/LED-1/history")
	if code != http.StatusOK {
		t.Fatalf("it answered %d: %v", code, body)
	}

	if body["text"] != "you: start it\norbit: started" {
		t.Errorf("it answered %v, want the conversation", body["text"])
	}

	if body["read"] != true {
		t.Errorf("it answered read=%v, want it to say the thread was read", body["read"])
	}
}

// TestABuildWithNoThreadBehindItSaysSoRatherThanFailing. The screen is one
// of several, and a page that refused to open because one port is nil would
// be a build losing a window over a feature it does not have.
func TestABuildWithNoThreadBehindItSaysSoRatherThanFailing(t *testing.T) {
	s := withThread([]view.Task{{ID: "LED-1", Band: view.ToDo, Repo: "ledger"}}, nil)

	code, body := ask(t, s, "/api/tasks/LED-1/history")
	if code != http.StatusOK {
		t.Fatalf("it answered %d: %v", code, body)
	}

	if body["read"] == true {
		t.Errorf("a build with no thread said it read one: %v", body)
	}

	if body["id"] != "LED-1" {
		t.Errorf("it answered about %v, want the task that was asked for", body["id"])
	}
}

// TestAThreadThatCouldNotBeReadSaysWhat. Nothing is hidden: an empty
// conversation and one that could not be read are different sentences, and
// the page says a different thing about each.
func TestAThreadThatCouldNotBeReadSaysWhat(t *testing.T) {
	s := withThread(
		[]view.Task{{ID: "LED-1", Band: view.ToDo, Repo: "ledger"}},
		aThread{err: errors.New("the record of LED-1 is not there")},
	)

	code, body := ask(t, s, "/api/tasks/LED-1/history")
	if code != http.StatusOK {
		t.Fatalf("it answered %d: %v", code, body)
	}

	if body["failed"] != "the record of LED-1 is not there" {
		t.Errorf("it answered %v, want what stopped it", body["failed"])
	}

	if body["read"] == true {
		t.Errorf("a thread that could not be read says it was: %v", body)
	}
}

// TestAThreadAboutATaskNothingKnowsIsNotFound, rather than an empty
// conversation about a task that does not exist.
func TestAThreadAboutATaskNothingKnowsIsNotFound(t *testing.T) {
	s := withThread(nil, aThread{text: "anything"})

	if code, _ := ask(t, s, "/api/tasks/NOPE-1/history"); code == http.StatusOK {
		t.Errorf("a task nothing knows about answered %d", code)
	}
}

// TestWhereAFlowCameFromIsSaidInAWord. The page groups the flows a reader
// wrote against the ones Orbit ships, so a name it could not place must not
// quietly become one of theirs.
func TestWhereAFlowCameFromIsSaidInAWord(t *testing.T) {
	cases := map[flow.Origin]string{
		flow.OriginBuiltin: "builtin",
		flow.OriginUser:    "yours",
		flow.OriginShadow:  "shadow",
		flow.OriginUnknown: "unknown",
	}

	for origin, want := range cases {
		if got := originName(origin); got != want {
			t.Errorf("a flow from %v is called %q, want %q", origin, got, want)
		}
	}

	// A classification this package has never heard of is unknown and not
	// one of the reader's own: a flow wrongly grouped as theirs is a flow
	// they will go looking for a file of.
	if got := originName(flow.Origin(99)); got != "unknown" {
		t.Errorf("a classification nothing recognises is called %q, want unknown", got)
	}
}
