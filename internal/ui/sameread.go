package ui

// Whether a reading that has just come back says anything new.
//
// The task view asks for the record and the files every half second, and
// every answer rebuilt all fifteen panes: up to a tenth of a second on a
// long run, four times a second, on the loop that answers the keys. Elio
// felt it as the window going heavy whenever something was running. Most
// of those answers are the same as the one before, and a pane built from
// the same reading is the same pane.

import (
	"slices"

	"github.com/e1i0r/orbit/internal/view"
)

// sameLog is whether two readings of a record are the same. A record only
// grows, so the same length ending on the same entry is the same record.
func sameLog(a, b []view.Entry) bool {
	if len(a) != len(b) {
		return false
	}

	if len(a) == 0 {
		return true
	}

	x, y := a[len(a)-1], b[len(b)-1]

	return x.At.Equal(y.At) && x.Kind == y.Kind && x.Phase == y.Phase && x.Text == y.Text
}

// sameFiles is whether two listings of a task's directory are the same.
func sameFiles(a, b []view.File) bool { return slices.Equal(a, b) }

// sameErr is whether two failures say the same thing, nil being one.
func sameErr(a, b error) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return a.Error() == b.Error()
}
