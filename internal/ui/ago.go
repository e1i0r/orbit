package ui

import "time"

// ago says how long ago a moment was, for a sentence rather than a column.
//
// It differs from elapsed in the first minute only. elapsed fills a
// seven-cell column where every row has to read as a number, so it counts
// seconds there; ago is written into prose, where the thing a person says
// about the last minute is "just now" and not "41s". Above a minute the two
// agree, and ago hands the answer to elapsed rather than restating its
// switch, so a unit that changes changes in one place.
//
// A zero time is not a moment and reads as the empty string, the way it does
// in elapsed: a task that has never run has no age to draw. A time in the
// future reads as "just now" — elapsed clamps a backwards clock rather than
// correcting it, and the first minute is where that clamp lands.
func ago(now, then time.Time) string {
	if then.IsZero() {
		return ""
	}

	if now.Sub(then) < time.Minute {
		return "just now"
	}

	return elapsed(now, then)
}
