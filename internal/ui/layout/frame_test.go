package layout

// The frame is where every row offset in the window comes from, and this is
// the table that holds it to account. The program this replaces spread the
// same arithmetic across several hundred literal offsets, so a pane that
// grew by a row broke a pane nobody had touched; the test that would have
// caught it is this one, and it needs no terminal to run.

import (
	"strings"
	"testing"
)

// regionsOf returns the four regions in the order they are drawn, so a test
// can walk them as a tiling rather than naming each one four times.
func regionsOf(f Frame) []struct {
	name string
	r    Strip
} {
	return []struct {
		name string
		r    Strip
	}{
		{"header", f.Header},
		{"status", f.Status},
		{"body", f.Body},
		{"band", f.Band},
		{"bar", f.Bar},
	}
}

// checkTiling asserts the one property the whole frame exists to hold: the
// regions start at row zero, follow one another with no gap and no overlap,
// cover exactly h rows, and none of them is negative. A negative height is
// the failure mode that matters — it is what an offset subtracted twice
// produces, and it draws as a pane that eats the one below it.
func checkTiling(t *testing.T, f Frame, w, h int) {
	t.Helper()

	y := 0

	for _, reg := range regionsOf(f) {
		if reg.r.H < 0 {
			t.Errorf("%s has height %d — no region may be negative", reg.name, reg.r.H)
		}

		if reg.r.Y != y {
			t.Errorf("%s starts at row %d, want %d — regions must tile with no gap or overlap",
				reg.name, reg.r.Y, y)
		}

		if reg.r.W != w {
			t.Errorf("%s is %d columns wide, want %d — every region spans the terminal",
				reg.name, reg.r.W, w)
		}

		y += reg.r.H
	}

	if y != h {
		t.Errorf("the five regions cover %d rows of %d — frame must tile height exactly", y, h)
	}
}

func TestFrameTilesEveryTerminalItAccepts(t *testing.T) {
	cases := []struct {
		name     string
		w, h     int
		wantBody int
	}{
		{"a large terminal", 200, 60, 50},
		{"a laptop", 120, 40, 30},
		{"a half screen", 100, 30, 20},
		{"the classic eighty by twenty-four", 80, 24, 14},
		{"the narrowest terminal orbit draws", 60, 20, 10},
		{"a shell pane with eight rows", 100, 8, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, err := Fit(c.w, c.h)
			if err != nil {
				t.Fatalf("Fit(%d, %d): %v", c.w, c.h, err)
			}

			checkTiling(t, f, c.w, c.h)

			if f.Body.H != c.wantBody {
				t.Errorf("body is %d rows, want %d — body is what is left after chrome",
					f.Body.H, c.wantBody)
			}

			if f.Body.H < 1 {
				t.Errorf("body is %d rows — a terminal this size must still show a task", f.Body.H)
			}
		})
	}
}

// TestFrameRefusesATerminalUnderTheMinimum is the rule the program this
// replaces did not have: it drew whatever fitted, so a narrow terminal got a
// crooked table instead of a sentence saying what was wrong.
func TestFrameRefusesATerminalUnderTheMinimum(t *testing.T) {
	f, err := Fit(59, 20)
	if err == nil {
		t.Fatalf("Fit(59, 20) = %+v, want an error naming the minimum", f)
	}

	if f != (Frame{}) {
		t.Errorf("Fit(59, 20) returned %+v beside its error — a refused frame has no regions", f)
	}

	for _, want := range []string{"60", "59"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Fit(59, 20) said %q, want %s — reader must know how far off they are",
				err.Error(), want)
		}
	}
}

// TestFrameStaysTotalOnAShortTerminal walks every height down to nothing.
// The window refuses a terminal narrower than MinWidth and says so, but it
// has no such rule about height: a two-row pane is a pane somebody has, and
// the frame has to answer for it without handing back a negative row count.
func TestFrameStaysTotalOnAShortTerminal(t *testing.T) {
	for h := 12; h >= 0; h-- {
		f, err := Fit(MinWidth, h)
		if err != nil {
			t.Fatalf("Fit(%d, %d): %v", MinWidth, h, err)
		}

		checkTiling(t, f, MinWidth, h)

		if h >= 6 && f.Body.H < 1 {
			t.Errorf("at height %d body is %d rows — body is last to give up a row", h, f.Body.H)
		}
	}
}

// TestFrameGivesUpTheBandBeforeTheBody states the surrender order as a
// table. The order is the decision: a status line is worth less than a row
// of tasks, and the header is worth more than either, because a window that
// cannot say what it is showing is worse than one showing less.
func TestFrameGivesUpTheBandBeforeTheBody(t *testing.T) {
	cases := []struct {
		h                               int
		header, status, body, band, bar int
	}{
		{11, 3, 2, 1, 3, 2},
		{10, 3, 2, 1, 2, 2},
		{9, 3, 2, 1, 2, 1},
		{8, 2, 2, 1, 2, 1},
		{7, 2, 1, 1, 2, 1},
		{6, 2, 0, 1, 2, 1},
		{5, 2, 0, 1, 1, 1},
		{4, 2, 0, 1, 0, 1},
		{3, 2, 0, 1, 0, 0},
		{2, 1, 0, 1, 0, 0},
		{1, 1, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}
	for _, c := range cases {
		f, err := Fit(MinWidth, c.h)
		if err != nil {
			t.Fatalf("Fit(%d, %d): %v", MinWidth, c.h, err)
		}

		got := []int{f.Header.H, f.Status.H, f.Body.H, f.Band.H, f.Bar.H}

		want := []int{c.header, c.status, c.body, c.band, c.bar}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("at height %d the regions are %v, want %v", c.h, got, want)
				break
			}
		}
	}
}

func TestRegionLineYHelpers(t *testing.T) {
	tall, err := Fit(MinWidth, 30)
	if err != nil {
		t.Fatalf("Fit tall: %v", err)
	}

	if tall.HeaderLineY() != 1 {
		t.Errorf("tall HeaderLineY = %d, want 1 (under top padding)", tall.HeaderLineY())
	}

	if tall.BandLineY() != tall.Band.Y+1 {
		t.Errorf("tall BandLineY = %d, want %d", tall.BandLineY(), tall.Band.Y+1)
	}

	if tall.BarLineY() != tall.Bar.Y {
		t.Errorf("tall BarLineY = %d, want %d", tall.BarLineY(), tall.Bar.Y)
	}

	short, err := Fit(MinWidth, 5)
	if err != nil {
		t.Fatalf("Fit short: %v", err)
	}

	if short.HeaderLineY() != 0 {
		t.Errorf("short HeaderLineY = %d, want 0", short.HeaderLineY())
	}

	if short.BandLineY() != short.Band.Y {
		t.Errorf("short BandLineY = %d, want %d", short.BandLineY(), short.Band.Y)
	}
}
