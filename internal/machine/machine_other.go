//go:build !darwin && !linux

package machine

// memoryUsed is not known here, and the queue does not hold runs back on a
// number it does not have.
func memoryUsed() (int, bool) { return 0, false }
