package store

// The queue's two limits, read the way every other setting is: a zero in
// the file is nobody having chosen, and reads as the shipped value.
//
// MaxRunning is how many runs go at once; more wait in the queue and start
// on their own as one finishes. MemoryCeiling is how full the machine's
// memory may be, as a percentage, for another to start; above it the next
// one waits until memory comes down.

// The queue's defaults: how many runs go at once, and how full memory may be
// for another to start.
const (
	defaultMaxRunning    = 3
	defaultMemoryCeiling = 85
)

// Running is how many runs the queue lets go at once, and three where
// nobody has chosen.
func (s Settings) Running() int {
	if s.MaxRunning <= 0 {
		return defaultMaxRunning
	}

	return s.MaxRunning
}

// MemoryBar is how full memory may be for another run to start, as a
// percentage, and eighty-five where nobody has chosen.
func (s Settings) MemoryBar() int {
	if s.MemoryCeiling <= 0 || s.MemoryCeiling > 100 {
		return defaultMemoryCeiling
	}

	return s.MemoryCeiling
}
