package task

// A task's work landing, written down where it happened.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestWhereTheMergeHappenedIsWrittenDownAndNotInferred.
//
// A branch disappears for three reasons and only one of them is delivery, so
// what a report counts as landed is something somebody did rather than
// something that stopped being there.
func TestWhereTheMergeHappenedIsWrittenDownAndNotInferred(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "the importer", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Merged(s, tk, "payments", "orbit/ACME-1"); err != nil {
		t.Fatalf("Merged: %v", err)
	}

	var found *record.Event

	for _, e := range eventsOf(t, s, tk) {
		if e.Kind == record.TaskMerged {
			found = &e
		}
	}

	if found == nil {
		t.Fatal("nothing in the record says the work landed")
	}

	if found.Data["repo"] != "payments" || found.Data["branch"] != "orbit/ACME-1" {
		t.Errorf("the record says it landed in %+v", found.Data)
	}
}
