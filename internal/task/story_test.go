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

	last := build(tk, f.Phases[1], true, nil, nil, nil, "", "", nil)
	if !strings.Contains(last, "## Story") {
		t.Errorf("the last phase is not asked for the story:\n%s", last)
	}

	first := build(tk, f.Phases[0], false, nil, nil, nil, "", "", nil)
	if strings.Contains(first, "## Story") {
		t.Errorf("a phase in the middle is asked for the story anyway:\n%s", first)
	}
}

// TestItIsTheLastPhaseOfTheFlowThatIsAsked.
//
// Which phase is the last one is worked out from where the run has got to,
// and nothing was reading it back: the prompts were built by hand with the
// answer handed in, and the record holds a story whether the phase was asked
// for one or not — a fake answers with what it was given either way.
//
// So it is read off a run: two phases, and only the second is asked.
func TestItIsTheLastPhaseOfTheFlowThatIsAsked(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-31", "the endpoint drops duplicates", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake(storyAnswer)

	f := flow.Flow{Name: "task", Phases: []flow.Phase{
		{Name: "1-plan", Engine: "fake"},
		{Name: "2-implement", Engine: "fake"},
	}}

	if err := Run(context.Background(), s, tk, f, fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 2 {
		t.Fatalf("the engine ran %d times, want both phases", len(fake.Calls))
	}

	if strings.Contains(fake.Calls[0].Prompt, "## Story") {
		t.Errorf("the first of two phases was asked for the story:\n%s", fake.Calls[0].Prompt)
	}

	if !strings.Contains(fake.Calls[1].Prompt, "## Story") {
		t.Errorf("the last phase was not asked for the story:\n%s", fake.Calls[1].Prompt)
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
	if err := Run(context.Background(), s, tk, f, fakes(engine.NewFake(storyAnswer)), nil); err != nil {
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

// TestWhatIsFedForwardIsBounded. The prompt goes on a command line, and
// Linux refuses a single argument over 128 KiB: a phase handed a megabyte of
// the phase before it died at exec with nothing in the record saying why.
func TestWhatIsFedForwardIsBounded(t *testing.T) {
	p := flow.Phase{Name: "2-implement", Engine: "fake", FeedOutput: true}

	fed := fedOutput(p, strings.Repeat("x", maxFed*3))
	if len(fed) > maxFed+200 {
		t.Errorf("a phase is fed %d bytes, want it cut to about %d", len(fed), maxFed)
	}

	if !strings.Contains(fed, "truncated") {
		t.Error("the cut is not admitted in what the phase is handed")
	}

	short := strings.Repeat("y", 100)
	if got := fedOutput(p, short); got != short {
		t.Error("an answer under the limit was changed on its way forward")
	}
}

// TestAStoryIsWholeOrItIsNot. Five fields or none: a story with a link
// missing draws as a chain that still looks whole, and the reader has no
// way to see which claim was never made.
func TestAStoryIsWholeOrItIsNot(t *testing.T) {
	full := Story{Entry: "a", Purpose: "b", Symptom: "c", Cause: "d", Fix: "e"}
	if !full.whole() {
		t.Error("five fields read as not whole")
	}

	if (Story{Entry: "a"}).whole() {
		t.Error("one field reads as whole")
	}

	if (Story{}).whole() {
		t.Error("nothing reads as whole")
	}
}

// TestTheStoryThatStandsIsTheLastOneTold.
//
// A task run three times told its story three times, and the two before it
// are about work that was thrown away. This is the door the panes read, so
// the one it answers with is the one a reader is shown.
func TestTheStoryThatStandsIsTheLastOneTold(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-29", "the endpoint drops duplicates", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Nothing has been run, so there is no story — and that is an answer.
	if got := StoryOf(s, tk); got != nil {
		t.Errorf("a task nobody has run carries the story %+v", got)
	}

	f := flow.Flow{Name: "task", Phases: []flow.Phase{{Name: "implement", Engine: "fake"}}}
	if err := Run(context.Background(), s, tk, f,
		fakes(engine.NewFake(storyAnswer)), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	first := StoryOf(s, tk)
	if first == nil {
		t.Fatal("a task that told its story carries none")
	}

	if first.Entry != "POST /items" || first.Cause != "the primary key collided" {
		t.Errorf("the story reads %+v, want the five fields the phase wrote", first)
	}

	// Run again, telling a different story: the one that stands is the one
	// about the work that is there.
	again := strings.ReplaceAll(storyAnswer, "POST /items", "PUT /items")
	if err := Run(context.Background(), s, tk, f, fakes(engine.NewFake(again)), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	last := StoryOf(s, tk)
	if last == nil || last.Entry != "PUT /items" {
		t.Errorf("the story that stands is %+v, want the one the second run told", last)
	}
}

// TestOnlyTheFiveFieldsAreReadOutOfAStory. The five are a closed list:
// anything else a phase writes under the heading is prose, and reading it as
// a field would put whatever a model invented into the shape a pane draws.
func TestOnlyTheFiveFieldsAreReadOutOfAStory(t *testing.T) {
	for _, one := range storyFields {
		if !known(one) {
			t.Errorf("%q is one of the five and does not read as one", one)
		}
	}

	for _, one := range []string{"", "entries", "purposes", "symptoms", "why", "ENTRY"} {
		if known(one) {
			t.Errorf("%q is not one of the five and reads as one", one)
		}
	}
}
