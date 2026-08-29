package board

import "time"

// Health is what the refresh loop knows about the state of the record right
// now: the state root, how many repositories and tasks are under it, how
// many bytes of events.jsonl they come to, what the last refresh read and
// how long it took.
//
// Every field is gathered from facts the refresh already touches. Nothing
// here costs a stat, a walk or an open of its own — a panel that charged
// for itself would be a panel nobody could afford to leave open, which is
// the one thing it is for.
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
