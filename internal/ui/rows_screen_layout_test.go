package ui

// detail_rows_screen_coverage_test.go covers three small, pure geometry
// functions this package leans on constantly but no rendering test happens
// to walk every branch of: paneHeight's own floor at a very short frame,
// follow and clampCursor's edges, and the frame primitives in screen.go.

import (
	"testing"
)

func TestPaneHeightFloor(t *testing.T) {
	tests := []struct {
		h, want int
	}{
		{10, 7},
		{4, 1},
		{3, 1},
		{2, 0},
		{0, 0},
	}
	for _, tt := range tests {
		if got := paneHeight(tt.h); got != tt.want {
			t.Errorf("paneHeight(%d) = %d, want %d", tt.h, got, tt.want)
		}
	}
}

func TestFollowDoesNotScrollAShortList(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.frame.Body.H = 1000 // taller than the fixture's whole row list
	m.cursor = 2

	got := m.follow()
	if got.offset != 0 {
		t.Errorf("follow with a body taller than the list = offset %d, want 0", got.offset)
	}
}

// TestFollowScrollsBothDirections is the rest of follow's job, once the
// list runs below the fold: a cursor past the visible window pulls the
// offset down after it, and a cursor above the window pulls it back up.
func TestFollowScrollsBothDirections(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	rows := m.rows()
	m.frame.Body.H = 3 // shorter than the fixture's row list

	m.offset = 0
	m.cursor = len(rows) - 1

	got := m.follow()
	if got.offset <= 0 {
		t.Errorf("follow with the cursor past the window left offset %d, want it to scroll down", got.offset)
	}

	m.offset = len(rows) - 1
	m.cursor = 0

	got = m.follow()
	if got.offset != 0 {
		t.Errorf("follow with the cursor above the window left offset %d, want 0", got.offset)
	}
}

func TestClampCursorOnAnEmptyList(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.filter = "no task anywhere matches this string"
	m.cursor = 5

	got := m.clampCursor()
	if got.cursor != 0 || got.offset != 0 {
		t.Errorf("clampCursor on an empty list = cursor %d offset %d, want both 0", got.cursor, got.offset)
	}
}

// TestClampCursorStepsOffABlankRow is the guard against a cursor that
// landed exactly on the separator line between two bands, which the cursor
// may never rest on.
func TestClampCursorStepsOffABlankRow(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	rows := m.rows()
	blankIdx := -1

	for i, r := range rows {
		if r.blank {
			blankIdx = i
			break
		}
	}

	if blankIdx < 0 {
		t.Fatal("the fixture board draws no blank separator row to test against")
	}

	m.cursor = blankIdx

	got := m.clampCursor()
	if got.cursor == blankIdx {
		t.Error("clampCursor left the cursor resting on a blank row")
	}
}

func TestBandRowsOneAndTwoTallRegions(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m.frame.Band.H = 1
	if rows := m.bandRows(); len(rows) != 1 {
		t.Errorf("bandRows with H=1 has %d rows, want 1", len(rows))
	}

	m.frame.Band.H = 2
	if rows := m.bandRows(); len(rows) != 2 {
		t.Errorf("bandRows with H=2 has %d rows, want 2", len(rows))
	}

	m.frame.Band.H = 0
	if rows := m.bandRows(); rows != nil {
		t.Errorf("bandRows with H=0 = %v, want nil", rows)
	}
}

func TestHeaderRowsOneAndTwoTallRegions(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m.frame.Header.H = 1
	if rows := m.headerRows(); len(rows) != 1 {
		t.Errorf("headerRows with H=1 has %d rows, want 1", len(rows))
	}

	m.frame.Header.H = 2
	if rows := m.headerRows(); len(rows) != 2 {
		t.Errorf("headerRows with H=2 has %d rows, want 2", len(rows))
	}

	m.frame.Header.H = 0
	if rows := m.headerRows(); rows != nil {
		t.Errorf("headerRows with H=0 = %v, want nil", rows)
	}
}

func TestBarRowsUsesThePaletteLineWhenOpen(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m.frame.Bar.H = 0
	if rows := m.barRows(); rows != nil {
		t.Errorf("barRows with H=0 = %v, want nil", rows)
	}

	m.frame.Bar.H = 1

	m.palette.open = true
	if rows := m.barRows(); len(rows) != 1 {
		t.Errorf("barRows with the palette open has %d rows, want 1", len(rows))
	}
}

func TestPageAndFitEdges(t *testing.T) {
	if got := page(0, 10, 0); got != 0 {
		t.Errorf("page(0, ...) = %d, want 0", got)
	}

	if got := page(5, 5, 0); got != 5 {
		t.Errorf("page with the offset already covering every row = %d, want 5", got)
	}

	if got := page(5, 20, 0); got != 4 {
		t.Errorf("page with more rows below the fold = %d, want one fewer than the region", got)
	}

	if got := fit("hello", 0); got != "" {
		t.Errorf("fit(..., 0) = %q, want empty", got)
	}

	if got := fit("hi", 10); got != "hi" {
		t.Errorf("fit within width = %q, want it unchanged", got)
	}

	if got := fit("a long sentence", 5); len([]rune(got)) > 5 {
		t.Errorf("fit truncated to %q, want at most 5 cells", got)
	}
}
