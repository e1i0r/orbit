package ui

// ago turns the distance between two moments into the words a reader gets.
// These tests pin the word for the first minute, the unit each magnitude
// lands in, the edges where one unit hands over to the next, and the two
// inputs that are not durations at all: a moment that never happened and a
// moment still ahead of the clock.

import (
	"testing"
	"time"
)

// The clock these cases are measured against. A fixed instant, so a failure
// is a fact about ago and not about when the test ran.
var agoNow = time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

func TestAgoPicksUnitByMagnitude(t *testing.T) {
	cases := []struct {
		name string
		back time.Duration
		want string
	}{
		{"the same instant", 0, "just now"},
		{"a few seconds", 3 * time.Second, "just now"},
		{"the last second before a minute", time.Minute - time.Second, "just now"},
		{"exactly one minute", time.Minute, "1m"},
		{"the minutes the task asks for", 3 * time.Minute, "3m"},
		{"the last minute before an hour", time.Hour - time.Minute, "59m"},
		{"exactly one hour", time.Hour, "1h"},
		{"the hours the task asks for", 2 * time.Hour, "2h"},
		{"the last hour before a day", 24*time.Hour - time.Hour, "23h"},
		{"exactly one day", 24 * time.Hour, "1d"},
		{"the days the task asks for", 5 * 24 * time.Hour, "5d"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ago(agoNow, agoNow.Add(-c.back)); got != c.want {
				t.Errorf("ago(now, now-%s) = %q, want %q", c.back, got, c.want)
			}
		})
	}
}

// A part of a unit is dropped rather than rounded up, so the number a reader
// sees is a count of whole units that have passed and never one that has not.
func TestAgoCountsWholeUnits(t *testing.T) {
	cases := []struct {
		name string
		back time.Duration
		want string
	}{
		{"most of the way to two minutes", 119 * time.Second, "1m"},
		{"most of the way to two hours", 2*time.Hour - time.Second, "1h"},
		{"most of the way to two days", 48*time.Hour - time.Second, "1d"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ago(agoNow, agoNow.Add(-c.back)); got != c.want {
				t.Errorf("ago(now, now-%s) = %q, want %q", c.back, got, c.want)
			}
		})
	}
}

// A zero time is not a moment. It reads as nothing rather than as the age of
// a clock that starts in year one.
func TestAgoReadsZeroTimeAsNothing(t *testing.T) {
	if got := ago(agoNow, time.Time{}); got != "" {
		t.Errorf("ago(now, zero time) = %q, want %q", got, "")
	}
}

// A record whose clock ran backwards reaches here as a moment in the future.
// It reads as the near past rather than as a negative number a caller would
// have to strip out of the middle of a sentence.
func TestAgoReadsTheFutureAsJustNow(t *testing.T) {
	for _, ahead := range []time.Duration{time.Second, time.Hour, 30 * 24 * time.Hour} {
		if got := ago(agoNow, agoNow.Add(ahead)); got != "just now" {
			t.Errorf("ago(now, now+%s) = %q, want %q", ahead, got, "just now")
		}
	}
}

// ago is a function of its two arguments and nothing else: the same pair has
// to answer the same string every time it is asked, so a row that is not
// being redrawn cannot drift from one that is.
func TestAgoIsPure(t *testing.T) {
	then := agoNow.Add(-90 * time.Minute)

	first, second := ago(agoNow, then), ago(agoNow, then)
	if first != second {
		t.Errorf("ago answered %q and then %q for the same pair", first, second)
	}
}
