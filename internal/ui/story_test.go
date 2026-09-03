package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

func storyEntry() view.Entry {
	return view.Entry{
		Kind: "task.story", At: ago(time.Minute),
		Story: &view.Story{
			Entry:   "POST /items",
			Purpose: "save the list Z in the database",
			Symptom: "repeated entries were silently not saved",
			Cause:   "the primary key collided",
			Fix:     "upsert instead of insert",
		},
	}
}

// TestTheOverviewDrawsTheStoryAsAChain. Five fields on five lines is a list;
// the point is that each one is the reason for the one under it, and the
// tree is the only shape that says so in eighty columns.
func TestTheOverviewDrawsTheStoryAsAChain(t *testing.T) {
	m, lines := onTab(t, tabOverview, []view.Entry{
		{Kind: "task.created", At: ago(time.Hour), Text: "Fix the save\n\nRepeated entries vanish."},
		storyEntry(),
	})
	_ = m

	text := ansi.Strip(strings.Join(lines, "\n"))
	for _, want := range []string{
		"POST /items",
		"save the list Z in the database",
		"repeated entries were silently not saved",
		"the primary key collided",
		"upsert instead of insert",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the overview does not carry %q:\n%s", want, text)
		}
	}

	if !strings.Contains(text, "└─") {
		t.Errorf("the story is not drawn as a chain:\n%s", text)
	}
}

// TestATaskWithNoStoryDrawsNoEmptyTree. Every task recorded before this
// existed has no story, and a heading over five blank rows is worse than the
// pane that was there before.
func TestATaskWithNoStoryDrawsNoEmptyTree(t *testing.T) {
	_, lines := onTab(t, tabOverview, []view.Entry{
		{Kind: "task.created", At: ago(time.Hour), Text: "Fix the save"},
	})

	if text := ansi.Strip(strings.Join(lines, "\n")); strings.Contains(text, "└─") {
		t.Errorf("a task with no story drew a chain anyway:\n%s", text)
	}
}

// TestTheNewestStoryIsTheOneDrawn. A task run twice told its story twice,
// and the one that counts is the one about the run that stands.
func TestTheNewestStoryIsTheOneDrawn(t *testing.T) {
	old := storyEntry()
	old.At = ago(time.Hour)
	old.Story = &view.Story{Entry: "GET /old", Purpose: "p", Symptom: "s", Cause: "c", Fix: "f"}

	_, lines := onTab(t, tabOverview, []view.Entry{old, storyEntry()})

	text := ansi.Strip(strings.Join(lines, "\n"))
	if strings.Contains(text, "GET /old") {
		t.Errorf("the overview drew the story of an older attempt:\n%s", text)
	}

	if !strings.Contains(text, "POST /items") {
		t.Errorf("the overview did not draw the newest story:\n%s", text)
	}
}
