package cli

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/task"
)

func aStory() *task.Story {
	return &task.Story{
		Entry:   `POST /items`,
		Purpose: `save the list "Z" in the database`,
		Symptom: "repeated entries were silently not saved",
		Cause:   "the primary key collided",
		Fix:     "upsert instead of insert",
	}
}

// TestThePullRequestCarriesTheSameStoryAsTheTerminal. One datum, two
// renders: the terminal can only draw text and a pull request can draw a
// diagram, and neither is allowed to know something the other does not.
func TestThePullRequestCarriesTheSameStoryAsTheTerminal(t *testing.T) {
	body := storySection(aStory())

	if !strings.Contains(body, "```mermaid") {
		t.Errorf("the pull request draws no diagram:\n%s", body)
	}

	for _, want := range []string{
		"POST /items",
		"save the list 'Z' in the database",
		"repeated entries were silently not saved",
		"the primary key collided",
		"upsert instead of insert",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the story section does not carry %q:\n%s", want, body)
		}
	}
}

// TestADiagramNeverCarriesAQuoteThatBreaksIt. Mermaid labels are quoted, so
// a quote inside one ends the label and the rest of the sentence becomes
// syntax — a diagram that fails to render on GitHub, silently, in the one
// place a reviewer was going to read it.
func TestADiagramNeverCarriesAQuoteThatBreaksIt(t *testing.T) {
	body := storySection(&task.Story{
		Entry: `the "items" route`, Purpose: `keep "Z"`, Symptom: "s", Cause: "c", Fix: "f",
	})

	_, diagram, found := strings.Cut(body, "```mermaid")
	if !found {
		t.Fatalf("the story section drew no diagram:\n%s", body)
	}

	if strings.Contains(diagram, `"the "items" route"`) {
		t.Errorf("a label carries a quote that closes it early:\n%s", diagram)
	}
}

// TestATaskWithNoStoryAddsNothingToItsPullRequest.
func TestATaskWithNoStoryAddsNothingToItsPullRequest(t *testing.T) {
	if got := storySection(nil); got != "" {
		t.Errorf("a task with no story added %q to its pull request", got)
	}
}

// TestTheBodyCarriesTheStoryWhenThereIsOne.
func TestTheBodyCarriesTheStoryWhenThereIsOne(t *testing.T) {
	body := bodyOf(task.Task{ID: "ACME-1", Text: "Fix the save"}, nil, aStory())
	if !strings.Contains(body, "```mermaid") || !strings.Contains(body, "ACME-1") {
		t.Errorf("the body does not carry both the task and its story:\n%s", body)
	}

	plain := bodyOf(task.Task{ID: "ACME-1", Text: "Fix the save"}, nil, nil)
	if strings.Contains(plain, "mermaid") {
		t.Errorf("a task with no story got a diagram anyway:\n%s", plain)
	}
}
