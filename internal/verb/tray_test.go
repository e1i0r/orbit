package verb

// The tray as a terminal reads it, and where a kept rule is said to have
// gone.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/learn"
)

// TestTheTrayIsNumberedFromOneAndLinesUp.
//
// The number is the position in this listing and nothing more durable than
// that: keep and drop read it back out of the same order, so a listing
// counted from anywhere else keeps the wrong sentence. The place is a column
// only when something has one, because most terminals here are a hundred
// columns across and the sentence is what somebody came to read.
func TestTheTrayIsNumberedFromOneAndLinesUp(t *testing.T) {
	at := time.Date(2026, 9, 18, 9, 30, 0, 0, time.UTC)

	got := numbered([]learn.Said{
		{At: at, Text: "amounts are cents", By: learn.Operator, Path: "internal/db"},
		{At: at, Text: "the commits are in English", By: learn.Operator},
	})

	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("two sentences listed as %d lines:\n%s", len(lines), got)
	}

	if !strings.HasPrefix(strings.TrimSpace(lines[0]), "1 ") {
		t.Errorf("the first row reads %q, want it numbered 1", lines[0])
	}

	if !strings.HasPrefix(strings.TrimSpace(lines[1]), "2 ") {
		t.Errorf("the second row reads %q, want it numbered 2", lines[1])
	}

	// The place is a column, so the sentence beside a row that has none
	// starts where the sentence beside a row that has one starts.
	for _, want := range []string{"internal/db", "amounts are cents", "the commits are in English"} {
		if !strings.Contains(got, want) {
			t.Errorf("the tray reads:\n%s\nwant it to carry %q", got, want)
		}
	}

	if at := strings.Index(lines[0], "amounts"); at != strings.Index(lines[1], "the commits") {
		t.Errorf("the sentences start at %d and %d, want one column:\n%s",
			at, strings.Index(lines[1], "the commits"), got)
	}
}

// TestATrayWithNoPlacesHasNoPlaceColumn, because a column of nothing is a
// gap the reader's eye crosses for no reason on a narrow terminal.
func TestATrayWithNoPlacesHasNoPlaceColumn(t *testing.T) {
	at := time.Date(2026, 9, 18, 9, 30, 0, 0, time.UTC)

	with := numbered([]learn.Said{{At: at, Text: "x", By: learn.Operator, Path: "internal/db"}})
	without := numbered([]learn.Said{{At: at, Text: "x", By: learn.Operator}})

	if strings.Index(without, "x") >= strings.Index(with, "x") {
		t.Errorf("a row with no place is no narrower than one with:\n%q\n%q", without, with)
	}
}

// TestWhereAKeptRuleWentIsWhatTheReaderIsTold.
//
// What the reader typed, then the folder the work was in when the sentence
// was said — and nothing at all for a rule about the whole checkout, because
// a dot is somebody saying so out loud rather than a place.
func TestWhereAKeptRuleWentIsWhatTheReaderIsTold(t *testing.T) {
	for _, one := range []struct {
		why    string
		typed  string
		worked string
		want   string
	}{
		{"what the reader typed wins", "internal/ui", "internal/db", "internal/ui"},
		{"and the folder the work was in when they typed nothing", "", "internal/db", "internal/db"},
		{"a dot is the whole checkout, which is no place", ".", "internal/db", ""},
		{"and a rule nobody put anywhere went nowhere", "", "", ""},
		{"whitespace is not a place", "   ", "internal/db", "internal/db"},
	} {
		if got := placed(one.typed, one.worked); got != one.want {
			t.Errorf("placed(%q, %q) = %q, want %q — %s", one.typed, one.worked, got, one.want, one.why)
		}
	}
}
