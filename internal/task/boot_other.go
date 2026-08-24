//go:build !darwin && !linux

package task

import "time"

// bootTime has no answer on a platform this package has not been taught.
// Saying so is the whole implementation: Alive falls back to asking only
// whether the pid exists, which is what it did everywhere before the two
// files beside this one were written.
func bootTime() (time.Time, bool) { return time.Time{}, false }
