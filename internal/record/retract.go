package record

// Taking back a line that was already written.

import "time"

// Stamp is the canonical way to name one line of a log.
//
// An event carries no id — the shape is flat and small on purpose, because a
// record that needs a schema migration to stay readable is not one you can
// still read in a year with `cat` — so the only thing that already
// distinguishes one line from another is when it was written. Nanoseconds
// are enough for that: two turns of a conversation are not typed in the same
// nanosecond, and two events appended by one process cannot share a clock
// reading.
//
// It is spelled out here, once, because a name that is written one way and
// read another names nothing.
func Stamp(at time.Time) string {
	return at.UTC().Format(time.RFC3339Nano)
}

// Retracted is the set of lines a later line took back, by Stamp.
//
// Taking back is writing, not erasing. The log is append-only and that is
// what makes it worth having: a turn somebody regretted is still a turn they
// took, and a reader looking at why the supervisor concluded something needs
// to see the sentence that was withdrawn as much as the ones that were not.
// What a retraction changes is what the line is still allowed to do — it
// stops being repeated into the next prompt — and not whether it happened.
func Retracted(events []Event) map[string]bool {
	var gone map[string]bool

	for _, e := range events {
		if e.Kind != SupervisorRetracted {
			continue
		}

		at := e.Data["at"]
		if at == "" {
			continue
		}

		if gone == nil {
			gone = map[string]bool{}
		}

		gone[at] = true
	}

	return gone
}
