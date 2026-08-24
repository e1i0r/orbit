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
			t.Errorf("%s starts at row %d, want %d — the regions must tile with no gap and no overlap", reg.name, reg.r.Y, y)
		}
		if reg.r.W != w {
			t.Errorf("%s is %d columns wide, want %d — every region spans the terminal", reg.name, reg.r.W, w)
		}
		y += reg.r.H
	}
	if y != h {
		t.Errorf("the four regions cover %d rows of %d — the frame must tile the height exactly", y, h)
	}
}

func TestFrameTilesEveryTerminalItAccepts(t *testing.T) {
	cases := []struct {
		name     string
		w, h     int
		wantBody int
	}{
		{"a large terminal", 200, 60, 55},
		{"a laptop", 120, 40, 35},
		{"a half screen", 100, 30, 25},
		{"the classic eighty by twenty-four", 80, 24, 19},
		{"the narrowest terminal orbit draws", 60, 20, 15},
		{"a shell pane with eight rows", 100, 8, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, err := Fit(c.w, c.h)
			if err != nil {
				t.Fatalf("Fit(%d, %d): %v", c.w, c.h, err)
			}
			checkTiling(t, f, c.w, c.h)
			if f.Body.H != c.wantBody {
				t.Errorf("body is %d rows, want %d — the body is what is left after the header, the band and the bar", f.Body.H, c.wantBody)
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
			t.Errorf("Fit(59, 20) said %q, want it to name %s — the reader has to know how far off they are", err.Error(), want)
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
			t.Errorf("at height %d the body is %d rows — the body is the last thing to give up a row", h, f.Body.H)
		}
	}
}

// TestFrameGivesUpTheBandBeforeTheBody states the surrender order as a
// table. The order is the decision: a status line is worth less than a row
// of tasks, and the header is worth more than either, because a window that
// cannot say what it is showing is worse than one showing less.
func TestFrameGivesUpTheBandBeforeTheBody(t *testing.T) {
	cases := []struct {
		h                       int
		header, body, band, bar int
	}{
		{8, 2, 3, 2, 1},
		{6, 2, 1, 2, 1},
		{5, 2, 1, 1, 1},
		{4, 2, 1, 0, 1},
		{3, 2, 1, 0, 0},
		{2, 1, 1, 0, 0},
		{1, 1, 0, 0, 0},
		{0, 0, 0, 0, 0},
	}
	for _, c := range cases {
		f, err := Fit(MinWidth, c.h)
		if err != nil {
			t.Fatalf("Fit(%d, %d): %v", MinWidth, c.h, err)
		}
		got := []int{f.Header.H, f.Body.H, f.Band.H, f.Bar.H}
		want := []int{c.header, c.body, c.band, c.bar}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("at height %d the regions are %v, want %v", c.h, got, want)
				break
			}
		}
	}
}
