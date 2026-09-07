package db

// Letting go of the record, and the one event it will not take.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestAnEventTooBigForOneRowIsRefused.
//
// An event this size is a run that has gone wrong — an engine looping on its
// own output — and the record is what every refresh of the board reads. One
// number for both records keeps them interchangeable: a state root filled
// from the files holds nothing this one would refuse.
func TestAnEventTooBigForOneRowIsRefused(t *testing.T) {
	d := open(t)

	huge := record.Event{
		Kind: record.PhaseToolCall,
		Text: strings.Repeat("x", record.MaxLine+1),
	}

	err := d.Append("ACME-1", huge)
	if err == nil {
		t.Fatal("an event over the line one row holds was written down")
	}

	if !strings.Contains(err.Error(), "over the") {
		t.Errorf("the refusal reads %q", err)
	}

	// And the task's log is still readable: the refusal cost nothing.
	events, err := d.Events("ACME-1")
	if err != nil {
		t.Fatalf("the log could not be read after the refusal: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("the refused event was written anyway: %+v", events)
	}
}

// TestTheRecordCanBeLetGoOfAndSaysWhereItWas.
func TestTheRecordCanBeLetGoOfAndSaysWhereItWas(t *testing.T) {
	d := open(t)

	if d.Path() == "" {
		t.Error("the record cannot say where it is")
	}

	if err := d.Close(); err != nil {
		t.Fatalf("closing the record: %v", err)
	}

	// Closed, it answers with an error rather than pretending to read.
	if _, err := d.Events("ACME-1"); err == nil {
		t.Error("a closed record answered a read")
	}
}

// TestAReaderIsGivenOnlyWhatArrivedAfterTheLastOneItSaw. A reader starts at
// zero and gets everything, which is the same cost as reading every log from
// the top and is paid once; after that most calls answer nothing at all.
func TestAReaderIsGivenOnlyWhatArrivedAfterTheLastOneItSaw(t *testing.T) {
	d := open(t)

	for _, id := range []string{"ACME-1", "ACME-2"} {
		if err := d.Append(id, record.Event{Kind: record.TaskCreated}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	all, err := d.Since(0, "")
	if err != nil {
		t.Fatalf("Since(0): %v", err)
	}

	if len(all) < 2 {
		t.Fatalf("a reader starting at zero was given %d changes", len(all))
	}

	last := all[len(all)-1].N

	more, err := d.Since(last, "")
	if err != nil {
		t.Fatalf("Since(%d): %v", last, err)
	}

	if len(more) != 0 {
		t.Errorf("a reader up to date was given %d changes", len(more))
	}

	// Asked about one task, it answers about that one only.
	one, err := d.Since(0, "ACME-2")
	if err != nil {
		t.Fatalf("Since(0, ACME-2): %v", err)
	}

	for _, c := range one {
		if c.Task != "ACME-2" {
			t.Errorf("a reader asking about ACME-2 was given %q", c.Task)
		}
	}
}
