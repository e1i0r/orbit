package tracker

// Reading what an issue actually says, and saying plainly when this machine
// cannot.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAnIssueThisMachineCannotReadComesBackAsItWasParsed, with the reason
// beside it: a caller can then write the task anyway and say what is
// missing, which is what the form does.
func TestAnIssueThisMachineCannotReadComesBackAsItWasParsed(t *testing.T) {
	t.Setenv("LINEAR_API_KEY", "")

	for _, iss := range []Issue{
		{Kind: "jira", ID: "ACME-1", Title: "the slug"},
		{Kind: "linear", ID: "PAY-71", Title: "the slug"},
	} {
		got, err := Read(context.Background(), iss)
		if !errors.Is(err, ErrNoKey) {
			t.Errorf("%s answered %v, want it to say there is no key", iss.Kind, err)
		}

		if got.ID != iss.ID || got.Title != iss.Title {
			t.Errorf("%s came back as %+v, want it as it was parsed", iss.Kind, got)
		}

		if got.Description != "" {
			t.Errorf("%s came back with a body it could not have read", iss.Kind)
		}
	}
}

// TestTheFormAsksWhetherThereIsAKeyBeforeItOffersToRead. A key for a tracker
// this program cannot read is not a key at all.
func TestTheFormAsksWhetherThereIsAKeyBeforeItOffersToRead(t *testing.T) {
	t.Setenv("LINEAR_API_KEY", "")

	if Readable("linear") {
		t.Error("a machine with no key says it can read Linear")
	}

	t.Setenv("LINEAR_API_KEY", "lin_api_whatever")

	if !Readable("linear") {
		t.Error("a machine with a key says it cannot read Linear")
	}

	if Readable("jira") || Readable("") {
		t.Error("a tracker this program has no credential for says it can be read")
	}
}

// TestAnIssueWithNoProviderOfItsOwnStillReadsAsATask. The prompt a run is
// handed has to say what the work is, where it came from, and what to do
// when the body is not in it.
func TestAnIssueWithNoProviderOfItsOwnStillReadsAsATask(t *testing.T) {
	got := defaultPrompt(Issue{
		Kind:   "asana",
		ID:     "ACME-1",
		Title:  "Retry the webhook on 5xx",
		RawURL: "https://app.asana.com/0/1/2",
	})

	for _, want := range []string{
		"Retry the webhook on 5xx",
		"ASANA",
		"ACME-1",
		"https://app.asana.com/0/1/2",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the prompt does not carry %q:\n%s", want, got)
		}
	}

	// With a body it carries that instead of telling somebody to go and
	// look: the body is the task.
	withBody := defaultPrompt(Issue{Kind: "asana", ID: "ACME-1", Description: "the real requirements"})
	if !strings.Contains(withBody, "the real requirements") {
		t.Errorf("the prompt drops the body it was given:\n%s", withBody)
	}
}

// TestWhatATrackerAnswersWithCannotDriveTheTerminal. An issue's title is
// drawn in the form and its body becomes the task: both come off somebody
// else's server, and a terminal reads an escape sequence in either as an
// instruction.
func TestWhatATrackerAnswersWithCannotDriveTheTerminal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		answer := `{"data":{"issue":{"title":"fix \u001b[2J the thing",` +
			`"description":"first \u001b[31mline\u001b[0m"}}}`
		if _, err := w.Write([]byte(answer)); err != nil {
			t.Errorf("the fake tracker could not answer: %v", err)
		}
	}))
	defer srv.Close()

	t.Setenv("LINEAR_API_KEY", "k")

	was := linearEndpoint
	linearEndpoint = srv.URL

	defer func() { linearEndpoint = was }()

	iss, err := Read(context.Background(), Issue{Kind: "linear", ID: "FRA-1"})
	if err != nil {
		t.Fatalf("read the issue: %v", err)
	}

	if strings.ContainsAny(iss.Title+iss.Description, "\x1b\x07") {
		t.Errorf("an issue carries control characters: %q / %q", iss.Title, iss.Description)
	}

	if !strings.Contains(iss.Title, "fix") || !strings.Contains(iss.Description, "line") {
		t.Errorf("taming the issue lost its words: %q / %q", iss.Title, iss.Description)
	}
}
