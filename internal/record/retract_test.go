package record

import (
	"testing"
	"time"
)

// TestStampIsTheSameNameFromEitherSide: an event has no id, so the whole
// scheme rests on one timestamp being written and read the same way. A
// retraction names a line by Stamp and a reader looks it up by Stamp; if the
// two ever disagree the retraction points at nothing.
func TestStampIsTheSameNameFromEitherSide(t *testing.T) {
	at := time.Date(2026, 8, 28, 9, 15, 0, 123456789, time.UTC)

	local := at.In(time.FixedZone("somewhere", 5*3600))
	if Stamp(at) != Stamp(local) {
		t.Errorf("Stamp differs by zone: %q vs %q", Stamp(at), Stamp(local))
	}
	// Two turns a nanosecond apart are two names.
	if Stamp(at) == Stamp(at.Add(time.Nanosecond)) {
		t.Error("a nanosecond apart, two lines got the same name")
	}
}

func TestRetractedCollectsWhatLaterLinesTookBack(t *testing.T) {
	first := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
	second := first.Add(time.Minute)

	gone := Retracted([]Event{
		{At: first, Kind: SupervisorMessage, Text: "the one I regret"},
		{At: second, Kind: SupervisorMessage, Text: "still standing"},
		{At: second.Add(time.Minute), Kind: SupervisorRetracted, Data: map[string]string{"at": Stamp(first)}},
	})
	if !gone[Stamp(first)] {
		t.Error("the retracted line is not in the set")
	}

	if gone[Stamp(second)] {
		t.Error("a line nobody took back is in the set")
	}
}

func TestRetractedIgnoresARetractionThatNamesNothing(t *testing.T) {
	gone := Retracted([]Event{
		{Kind: SupervisorRetracted},
		{Kind: SupervisorRetracted, Data: map[string]string{"at": ""}},
	})
	if len(gone) != 0 {
		t.Errorf("Retracted = %v, want nothing: neither line names a turn", gone)
	}

	if Retracted(nil) == nil {
		return // a nil set reads false for every key, which is the answer.
	}
}
