package db

// The size one row holds, from both sides of the line.
//
// The cap is tested from both sides, because a check one byte too strict
// refuses the largest event the record can hold and says nothing about it —
// and because the number the refusal names is the one a person compares
// against the cap beside it.

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// atExactly is an event whose JSON is exactly n bytes, padded out in Text.
func atExactly(t *testing.T, n int) record.Event {
	t.Helper()

	e := record.Event{Kind: record.TaskCreated, At: time.Now().UTC()}

	bare, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("encode the empty event: %v", err)
	}

	e.Text = "x"

	one, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("encode the event with a character in it: %v", err)
	}

	// The field name and its quotes, without the character itself.
	overhead := len(one) - len(bare) - 1
	e.Text = strings.Repeat("x", n-len(bare)-overhead)

	line, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("encode the padded event: %v", err)
	}

	if len(line) != n {
		t.Fatalf("the event is %d bytes, want exactly %d", len(line), n)
	}

	return e
}

// TestTheLastRowThatFitsIsTaken. The cap was only ever tested from above, so
// a check one byte too strict would have refused the largest event the
// record can actually hold and nothing would have said so.
func TestTheLastRowThatFitsIsTaken(t *testing.T) {
	d := open(t)

	// One under, because the newline the file log writes counts: an event
	// of exactly MaxLine bytes is a line of MaxLine+1 and is refused.
	e := atExactly(t, record.MaxLine-1)

	if err := tooBig(e); err != nil {
		t.Errorf("the largest event one row holds was refused: %v", err)
	}

	if err := d.Append("ACME-1", e); err != nil {
		t.Errorf("the largest event one row holds was not written: %v", err)
	}

	events, err := d.Events("ACME-1")
	if err != nil {
		t.Fatalf("read it back: %v", err)
	}

	if len(events) != 1 || len(events[0].Text) != len(e.Text) {
		t.Errorf("it came back as %d events of %d characters, want the one it was",
			len(events), len(events[0].Text))
	}
}

// TestTheRefusalSaysHowBigItWas. The number is the line the log would have
// written — the JSON and its newline — because that is the number a person
// compares against the one beside it, and a refusal naming a size nothing
// measured sends them looking for an event that does not exist.
func TestTheRefusalSaysHowBigItWas(t *testing.T) {
	e := atExactly(t, record.MaxLine)

	err := tooBig(e)
	if err == nil {
		t.Fatal("an event over the cap was accepted")
	}

	// MaxLine+1: the JSON is exactly MaxLine and the newline is the byte
	// that puts it over.
	if !strings.Contains(err.Error(), strconv.Itoa(record.MaxLine+1)) {
		t.Errorf("the refusal is %q, want it to name %d bytes", err, record.MaxLine+1)
	}

	if !strings.Contains(err.Error(), strconv.Itoa(record.MaxLine)) {
		t.Errorf("the refusal is %q, want it to name the %d cap it is over", err, record.MaxLine)
	}
}
