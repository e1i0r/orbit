package board

// Health is the state of the .jsonl database and the store: the root,
// repositories and task counts, bytes of events.jsonl across all of them,
// events read in the last refresh, the most recent write seen, and how long
// the last refresh took.
//
// Every field is gathered during the refresh from facts the refresh already
// touches — no additional stat, walk or open is performed.

import "time"

// Health is what the store and the refresh loop know about the state of the
// system right now, without making any additional syscalls.
type Health struct {
	// Root is the state root ($ORBIT_HOME).
	Root string

	// Repos is how many repositories were found under the workspace root.
	Repos int

	// Tasks is how many task records are held under the state root.
	Tasks int

	// Bytes is the total bytes of events.jsonl across all tasks.
	Bytes int64

	// EventsRead is how many events were read in the last refresh cycle.
	EventsRead int

	// LastWrite is the newest modification time seen on an events.jsonl file.
	LastWrite time.Time

	// Duration is how long the last refresh took.
	Duration time.Duration

	// Errs is how many errors occurred during the last refresh.
	Errs int
}
