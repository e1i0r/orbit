package task

// The delivery verbs are written down where the key is pressed, and the
// whole point of them is the gap between the two events: a verb that is
// asked for and not yet answered is what a reader sees while an engine is
// off doing it.

import (
	"errors"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

func TestDeliveringRecordsTheVerbAndWhatWasHandedIt(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DEL-1", "delivery test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Delivering(s, tk, "FIX CHECKS", "supervisor"); err != nil {
		t.Fatalf("Delivering: %v", err)
	}

	e := lastEvent(t, s, tk)
	if e.Kind != record.DeliverAsked {
		t.Errorf("kind = %q, want %q", e.Kind, record.DeliverAsked)
	}

	if e.Data["verb"] != "FIX CHECKS" {
		t.Errorf("verb = %q, want the caption the key was offered under", e.Data["verb"])
	}

	if e.Data["by"] != "supervisor" {
		t.Errorf("by = %q, want what was handed the work", e.Data["by"])
	}
}

func TestDeliveredKeepsWhatCameBack(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DEL-2", "delivery test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Delivered(s, tk, "CREATE PR", "opened https://example.test/pr/1", nil); err != nil {
		t.Fatalf("Delivered: %v", err)
	}

	e := lastEvent(t, s, tk)
	if e.Kind != record.DeliverAnswered {
		t.Errorf("kind = %q, want %q", e.Kind, record.DeliverAnswered)
	}

	if e.Text != "opened https://example.test/pr/1" {
		t.Errorf("text = %q, want what came back", e.Text)
	}

	if _, ok := e.Data["error"]; ok {
		t.Errorf("error = %q, want nothing on a verb that worked", e.Data["error"])
	}
}

// A verb that broke is still an answer: it came back, and why it broke is
// the only thing a reader can act on.
func TestDeliveredCarriesTheFailure(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DEL-3", "delivery test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Delivered(s, tk, "MERGE PR", "", errors.New("the branch is behind")); err != nil {
		t.Fatalf("Delivered: %v", err)
	}

	e := lastEvent(t, s, tk)
	if e.Data["error"] != "the branch is behind" {
		t.Errorf("error = %q, want why it broke", e.Data["error"])
	}
}

// Neither of them is written without the verb it is about. An event that
// says a delivery happened and not which one is a row the flow tree cannot
// draw and a reader cannot read.
func TestADeliveryNeedsItsVerb(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DEL-4", "delivery test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Delivering(s, tk, "  ", "supervisor"); err == nil {
		t.Error("Delivering with no verb = nil, want a refusal")
	}

	if err := Delivered(s, tk, "", "done", nil); err == nil {
		t.Error("Delivered with no verb = nil, want a refusal")
	}
}
