package view

// The task story as the log reads it.
//
// storyOf is where "is this story whole" is answered, and it is answered here
// rather than in the drawing because two panes asking it on every frame are
// two panes that will one day answer it differently. The assertion worth
// making is the second one: a story that arrived with a field missing is not
// a story drawn with a hole in it, it is no story at all.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestAStoryOfAnotherKindIsNoStory.
func TestAStoryOfAnotherKindIsNoStory(t *testing.T) {
	if got := storyOf(record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"}); got != nil {
		t.Errorf("an event that is not a story read as one: %+v", got)
	}
}

// TestAWholeStoryIsRead.
func TestAWholeStoryIsRead(t *testing.T) {
	got := storyOf(record.Event{
		Kind: record.TaskStory,
		Data: map[string]string{
			"entry":   "webhook.send",
			"purpose": "deliver the event",
			"symptom": "a 5xx dropped it",
			"cause":   "no retry",
			"fix":     "retry three times",
		},
	})

	if got == nil {
		t.Fatal("a story with all five fields was read as no story")
	}

	if got.Entry != "webhook.send" || got.Fix != "retry three times" {
		t.Errorf("the story came back as %+v", got)
	}
}

// TestAStoryWithAFieldMissingIsNoStory.
//
// Not a story with a hole in it. A pane drawing four of the five fields would
// be drawing a claim the record does not support, and the reader cannot tell
// which field is missing from the shape of what is on screen.
func TestAStoryWithAFieldMissingIsNoStory(t *testing.T) {
	for _, missing := range []string{"entry", "purpose", "symptom", "cause", "fix"} {
		fields := map[string]string{
			"entry":   "webhook.send",
			"purpose": "deliver the event",
			"symptom": "a 5xx dropped it",
			"cause":   "no retry",
			"fix":     "retry three times",
		}
		delete(fields, missing)

		if got := storyOf(record.Event{Kind: record.TaskStory, Data: fields}); got != nil {
			t.Errorf("a story without %q was read as one: %+v", missing, got)
		}
	}
}

// TestAStoryWithNoDataAtAllIsNoStory.
func TestAStoryWithNoDataAtAllIsNoStory(t *testing.T) {
	if got := storyOf(record.Event{Kind: record.TaskStory}); got != nil {
		t.Errorf("a story that arrived with nothing in it read as one: %+v", got)
	}
}
