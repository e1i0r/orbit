package mcp

// The readings the inspector hands a model: one entry per event, and the
// tail of the last thing an engine printed.
//
// A model cannot scroll. What it is given is all it will ever see of the
// run, so a key holding the empty string reads as a fact that is there and
// says nothing, and a tail that does not say it is a tail reads as the whole
// of what happened.

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/record"
)

// TestATimelineEntrySaysOnlyWhatTheEventHas.
func TestATimelineEntrySaysOnlyWhatTheEventHas(t *testing.T) {
	at := time.Date(2026, 9, 18, 9, 30, 0, 0, time.UTC)

	got := timelineOf([]record.Event{
		{Kind: record.TaskCreated, At: at},
		{Kind: record.PhaseThought, At: at, Phase: "implement", Text: "read the file first"},
	})

	if len(got) != 2 {
		t.Fatalf("two events read as %d entries", len(got))
	}

	// Every entry says when and what, because those are the two an entry
	// cannot be without.
	for i, e := range got {
		if e["at"] != at || e["kind"] == "" {
			t.Errorf("entry %d is %v", i, e)
		}
	}

	bare := got[0]
	if _, there := bare["phase"]; there {
		t.Errorf("an event that happened in no phase carries phase=%v", bare["phase"])
	}

	if _, there := bare["text"]; there {
		t.Errorf("an event that said nothing carries text=%v", bare["text"])
	}

	full := got[1]
	if full["phase"] != "implement" || full["text"] != "read the file first" {
		t.Errorf("an event with both reads as %v", full)
	}
}

// TestWhetherTheLastOutputIsTheWholeOfIt.
//
// The flag is the only thing telling a model that what it is reading is a
// tail. Said wrongly, it stops looking for the rest of an answer that was
// cut — and asks its next question about the half it was shown.
func TestWhetherTheLastOutputIsTheWholeOfIt(t *testing.T) {
	ended := func(text string) []record.Event {
		return []record.Event{
			{Kind: record.TaskCreated},
			{Kind: record.PhaseFinished, Phase: "implement", Text: text},
		}
	}

	whole := lastOutputOf(ended(strings.Repeat("a", outputChars)))
	if whole == nil {
		t.Fatal("a phase that printed something answered nothing")
	}

	if whole["complete"] != true {
		t.Errorf("output of exactly %d characters reads as cut", outputChars)
	}

	kept, isText := whole["text"].(string)
	if !isText {
		t.Fatalf("the output came back as %T", whole["text"])
	}

	if kept != strings.Repeat("a", outputChars) {
		t.Errorf("what was kept is %d characters", utf8.RuneCountInString(kept))
	}

	over := lastOutputOf(ended(strings.Repeat("a", outputChars+1)))
	if over["complete"] != false {
		t.Errorf("output one character over reads as the whole of it")
	}

	// A phase that ended saying nothing is not the last output: there is
	// nothing to be the tail of, and an entry holding an empty string reads
	// as an engine that printed silence.
	if got := lastOutputOf(ended("")); got != nil {
		t.Errorf("a phase that printed nothing reads as %v", got)
	}

	// And a run with no ending at all has no last output.
	if got := lastOutputOf([]record.Event{{Kind: record.PhaseStarted, Phase: "implement"}}); got != nil {
		t.Errorf("a run that never ended reads as %v", got)
	}
}
