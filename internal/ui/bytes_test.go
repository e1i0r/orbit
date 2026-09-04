package ui

// FormatBytes turns a byte count into the size a person would say out loud.
// These tests pin the unit each magnitude lands in, the edges where one unit
// hands over to the next, and the two answers that are not sizes at all: a
// negative count and anything past the last unit we name.

import "testing"

func TestFormatBytesPicksUnitByMagnitude(t *testing.T) {
	cases := []struct {
		name string
		in   int64
		want string
	}{
		{"zero", 0, "0 B"},
		{"a few bytes", 512, "512 B"},
		{"last whole byte count", 1023, "1023 B"},
		{"exactly one kilobyte", 1024, "1.0 KB"},
		{"half way up the kilobytes", 1536, "1.5 KB"},
		{"a count that rounds up into megabytes", 1024*1024 - 1, "1.0 MB"},
		{"exactly one megabyte", 1024 * 1024, "1.0 MB"},
		{"a few megabytes", 5 * 1024 * 1024, "5.0 MB"},
		{"a count that rounds up into gigabytes", 1024*1024*1024 - 1, "1.0 GB"},
		{"exactly one gigabyte", 1024 * 1024 * 1024, "1.0 GB"},
		{"two and a half gigabytes", 2560 * 1024 * 1024, "2.5 GB"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FormatBytes(c.in); got != c.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// A terabyte has no unit of its own here, so it stays in gigabytes rather
// than inventing a name the caller never asked for.
func TestFormatBytesStopsAtGigabytes(t *testing.T) {
	const terabyte = 1024 * 1024 * 1024 * 1024
	if got, want := FormatBytes(terabyte), "1024.0 GB"; got != want {
		t.Errorf("FormatBytes(%d) = %q, want %q", int64(terabyte), got, want)
	}
}

// A negative count is not a size. It reads as zero rather than as a minus
// sign a caller would have to strip back out of the middle of a layout.
func TestFormatBytesReadsNegativeAsZero(t *testing.T) {
	if got, want := FormatBytes(-1), "0 B"; got != want {
		t.Errorf("FormatBytes(-1) = %q, want %q", got, want)
	}
}

// FormatBytes is layout, not state: the same count has to render the same
// string every time it is asked, the way FormatLatency does.
func TestFormatBytesIsPure(t *testing.T) {
	first, second := FormatBytes(4096), FormatBytes(4096)
	if first != second {
		t.Errorf("FormatBytes(4096) rendered %q and then %q", first, second)
	}
}
