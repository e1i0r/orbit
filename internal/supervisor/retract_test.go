package supervisor

// Taking back one turn of the supervisor thread.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// TestRetractSupervisorStopsRepeatingATurnWithoutErasingIt.
//
// The thread is append-only and the whole of it goes into every later
// prompt, so without a retraction one sentence sent by mistake would keep
// steering the supervisor for the life of the state directory. What a
// retraction changes is what the line is still allowed to
// do, not whether it happened.
func TestRetractSupervisorStopsRepeatingATurnWithoutErasingIt(t *testing.T) {
	s := fixture(t)
	for _, text := range []string{"first thing", "the one I regret", "third thing"} {
		if err := Record(s, "", "elio", "cli", "", "", text); err != nil {
			t.Fatalf("Record %q: %v", text, err)
		}
	}

	events, err := Events(s)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if err := Retract(s, events[1].At); err != nil {
		t.Fatalf("Retract: %v", err)
	}

	events, err = Events(s)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if len(events) != 4 {
		t.Fatalf("the log holds %d events, want 4: three turns and the line that takes one back", len(events))
	}

	if events[1].Text != "the one I regret" {
		t.Errorf("the retracted turn is gone from the log; events[1] = %+v", events[1])
	}

	got := history(events)
	if strings.Contains(got, "the one I regret") {
		t.Errorf("the retracted turn is still in the prompt: %q", got)
	}

	if !strings.Contains(got, "first thing") || !strings.Contains(got, "third thing") {
		t.Errorf("a retraction took more than its own line with it: %q", got)
	}

	if strings.Contains(got, record.SupervisorRetracted) {
		t.Errorf("the retraction itself became a turn of the conversation: %q", got)
	}
}

// TestRetractSupervisorRefusesWhatItCannotFind: a retraction that matches no
// line is a typo, and accepting one leaves somebody believing they took
// something back.
func TestRetractSupervisorRefusesWhatItCannotFind(t *testing.T) {
	s := fixture(t)
	if err := Retract(nil, time.Now()); err == nil {
		t.Error("Retract on a nil store answered nil, want error")
	}

	if err := Retract(s, time.Time{}); err == nil {
		t.Error("Retract with no timestamp answered nil, want error")
	}

	if err := Retract(s, time.Now()); err == nil {
		t.Error("Retract over an empty thread answered nil, want error")
	}

	if err := Record(s, "", "elio", "cli", "", "", "the only turn"); err != nil {
		t.Fatalf("Record: %v", err)
	}

	events, err := Events(s)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if err := Retract(s, events[0].At.Add(time.Nanosecond)); err == nil {
		t.Error("Retract a nanosecond off the turn answered nil, want error")
	}

	// And a retraction is not a turn: there is nothing to take back about it.
	if err := Retract(s, events[0].At); err != nil {
		t.Fatalf("Retract: %v", err)
	}

	events, err = Events(s)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if err := Retract(s, events[1].At); err == nil {
		t.Error("retracting a retraction answered nil, want error")
	}
}
