package ui

import (
	"fmt"
	"math"
)

// byteUnits are the names FormatBytes is allowed to use, in the order it
// climbs them. The list stops at GB on purpose: a size this program shows a
// person is a log or a diff, and naming terabytes would be naming a scale
// nothing here produces.
var byteUnits = [...]string{"KB", "MB", "GB"}

// FormatBytes turns a byte count into the size a person would say out loud:
// whole bytes below a kilobyte, and one decimal place above it, because the
// second decimal is noise at every size a reader is scanning past. A count
// past the last unit stays in gigabytes rather than inventing a name, and a
// negative count reads as zero — it is not a size, and a minus sign buried
// in a layout is worse than the zero it stands for. Pure layout: bytes in,
// string out, nothing consulted in between.
//
// This is not formatBytes in pane_artifacts.go, which is the artifacts pane
// writing a file size into a narrow column and reads "2 k" where this reads
// "2.0 KB". Callers outside that column want this one.
func FormatBytes(b int64) string {
	if b < 1024 {
		if b < 0 {
			b = 0
		}
		return fmt.Sprintf("%d B", b)
	}
	size, unit := float64(b)/1024, byteUnits[0]
	for _, next := range byteUnits[1:] {
		// The rounded size decides the unit, not the raw one: a count a
		// hair under a megabyte rounds to 1024.0 at one decimal place, and
		// "1024.0 KB" is a size no one writes.
		if math.Round(size*10)/10 < 1024 {
			break
		}
		size, unit = size/1024, next
	}
	return fmt.Sprintf("%.1f %s", size, unit)
}
