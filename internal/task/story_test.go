package task

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

const storyAnswer = `Fixed it.

## Story

entry: POST /items
purpose: save the list Z in the database
symptom: repeated entries were silently not saved
cause: the primary key collided
fix: upsert instead of insert
`

// TestTheStoryIsReadOutOfTheLastPhasesAnswer. The five fields are what the
// report tab could never assemble: no single source holds them, and the one
// that can say all five is the engine that did the work.
func TestTheStoryIsReadOutOfTheLastPhasesAnswer(t *testing.T) {
	got, ok := storyIn(storyAnswer)
	if !ok {
		t.Fatal("no story was read out of an answer that told one")
	}

	for field, want := range map[string]string{
		"entry":   "POST /items",
		"purpose": "save the list Z in the database",
		"symptom": "repeated entries were silently not saved",
		"cause":   "the primary key collided",
		"fix":     "upsert instead of insert",
	} {
		if got[field] != want {
			t.Errorf("%s = %q, want %q", field, got[field], want)
		}
	}
}

// TestAnAnswerWithoutTheFiveFieldsTellsNoStory. Half a story on the overview
// tab is worse than none: it is a shape that looks authoritative with a
// field missing, and the reader cannot tell which.
func TestAnAnswerWithoutTheFiveFieldsTellsNoStory(t *testing.T) {
	if _, ok := storyIn("## Story\n\nentry: POST /items\nfix: upsert\n"); ok {
		t.Error("a story was read out of two fields of five")
	}

	if _, ok := storyIn("I did the work and it went fine."); ok {
		t.Error("a story was read out of an answer that told none")
	}
}

// TestTheLastPhaseIsAskedForTheStory, and only the last: the story is about
// the task, and a phase in the middle of one does not know how it ends.
func TestTheLastPhaseIsAskedForTheStory(t *testing.T) {
	tk := Task{ID: "ACME-1", Text: "fix the save"}
	f := flow.Flow{Name: "task", Phases: []flow.Phase{
		{Name: "1-plan", Engine: "fake"},
		{Name: "2-implement", Engine: "fake"},
	}}

	last := promptFor(tk, f, 2, nil, nil, "", nil)
	if !strings.Contains(last, "## Story") {
		t.Errorf("the last phase is not asked for the story:\n%s", last)
	}

	first := promptFor(tk, f, 1, nil, nil, "", nil)
	if strings.Contains(first, "## Story") {
		t.Errorf("a phase in the middle is asked for the story anyway:\n%s", first)
	}
}

// TestAFinishedTaskCarriesItsStory.
func TestAFinishedTaskCarriesItsStory(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-28", "the endpoint drops duplicates", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := flow.Flow{Name: "task", Phases: []flow.Phase{{Name: "implement", Engine: "fake"}}}
	if err := Run(context.Background(), s, tk, f, map[string]engine.Engine{"fake": engine.NewFake(storyAnswer)}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	var story record.Event

	for _, e := range events {
		if e.Kind == record.TaskStory {
			story = e
		}
	}

	if story.Kind == "" {
		t.Fatalf("the record holds no story: %v", kindsOf(events))
	}

	if story.Data["entry"] != "POST /items" || story.Data["cause"] != "the primary key collided" {
		t.Errorf("the story carries %v, want the five fields the phase wrote", story.Data)
	}
}
