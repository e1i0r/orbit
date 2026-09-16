package task

// The lines the record keeps about a task changing hands, and the two it
// keeps about a task that could not.
//
// Apart from relay.go because that file decides and this one writes. What is
// decided is one question — who takes this on — and what is written is three
// different sentences for three different readers: one for somebody judging a
// finished task, one for somebody being asked to choose, and one for somebody
// deciding whether to wait.

import (
	"fmt"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// relayed is the line the record keeps about one engine handing a task to
// another.
func relayed(phase, from, to string) record.Event {
	return record.Event{
		Kind:  record.TaskRelayed,
		Phase: phase,
		Text: fmt.Sprintf("%s ran out in phase %q, and %s took the task on from there.",
			from, phase, to),
		Data: map[string]string{"from": from, "to": to, "phase": phase, "why": whyRanOut},
	}
}

// The two reasons a task changes engine. One is Orbit's doing and one is
// yours, and the record says which — a reader judging a task that passed
// through three engines is owed the difference.
const (
	whyRanOut = "ran out"
	whyChosen = "you chose it"
)

// chose is the line the record keeps about a person changing the engine, so
// that a task started again under another engine reads as a relay and not as
// a run that has nothing to do with the one before it.
func chose(phase, from, to string) record.Event {
	return record.Event{
		Kind:  record.TaskRelayed,
		Phase: phase,
		Text:  fmt.Sprintf("you handed the task from %s to %s.", from, to),
		Data:  map[string]string{"from": from, "to": to, "phase": phase, "why": whyChosen},
	}
}

// needsEngine ends a run that could have carried on, and says under what.
//
// Not a failure. Nothing is broken, the work stands, and what it is waiting
// for is a person to pick one of the names in the event — which is the whole
// difference between this and the run that has nowhere to go.
func needsEngine(s *store.Store, t Task, p flow.Phase, from string, free []string) error {
	line := fmt.Sprintf("%s ran out in phase %q. %s could take it on.",
		from, p.Name, strings.Join(free, ", "))

	//nolint:errcheck // best-effort: the caller's error is what matters, see failed
	_ = emit(s, t, record.Event{
		Kind:  record.TaskNeedsEngine,
		Phase: p.Name,
		Text:  line,
		Data:  map[string]string{"from": from, "phase": p.Name, "engines": strings.Join(free, ",")},
	})

	return fmt.Errorf("task %s, phase %q: %s", t.ID, p.Name, line)
}

// spellOut is a wait as a person says one: minutes, and no trailing seconds
// that were rounded away anyway.
//
// time.Duration prints "1h35m0s", which is a machine talking. The seconds are
// noise in a sentence about waiting an hour and a half, and a wait shorter
// than a minute is not a number at all.
func spellOut(d time.Duration) string {
	if d < time.Minute {
		return "less than a minute"
	}

	said := strings.TrimSuffix(d.Round(time.Minute).String(), "0s")
	if strings.HasSuffix(said, "h0m") {
		said = strings.TrimSuffix(said, "0m")
	}

	return said
}

// noEngine ends a run that has nowhere to go, and says until when.
//
// Until when is the point of it. Allowances come back, so an hour is an
// answer and "abandoned" is not — and when nothing on this machine can say
// what the hour is, it says that instead of naming one.
func noEngine(s *store.Store, t Task, p flow.Phase, from string, back time.Duration) error {
	line := fmt.Sprintf("%s ran out in phase %q, and no engine has anything left", from, p.Name)
	data := map[string]string{"from": from, "phase": p.Name}

	if back > 0 {
		line += " for another " + spellOut(back)
		// The exact duration and not the spelled one: the line is for a
		// person and the field is for whatever reads it next, and a reader
		// that had to parse "an hour and a half" would be parsing prose.
		data["back"] = back.String()
	} else {
		line += ", and nothing here can say when one comes back"
	}

	//nolint:errcheck // best-effort: the caller's error is what matters, see failed
	_ = emit(s, t, record.Event{
		Kind:  record.TaskNoEngine,
		Phase: p.Name,
		Text:  line + ".",
		Data:  data,
	})

	return fmt.Errorf("task %s: %s", t.ID, line)
}
