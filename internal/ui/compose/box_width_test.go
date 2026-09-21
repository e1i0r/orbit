package compose

// The form a task is written in, drawn at every width it has room for.
//
// Every box on this screen is built by subtracting: the border from the
// window, the label from the border, the padding from what is left. A sum
// that is one out draws a box whose right-hand edge steps in and out as the
// reader moves between fields, and nobody sees it at the width it was
// written at.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/layout"
)

// atWidth is the form on a window of the given size.
func atWidth(t *testing.T, w, h int) (State, Env) {
	t.Helper()

	s, e := form(t)

	frame, err := layout.Fit(w, h)
	if err != nil {
		t.Skipf("a window of %dx%d is too small to draw in: %v", w, h, err)
	}

	e.Frame = frame

	return s, e
}

// TestNothingOnTheFormRunsOffTheWindow.
//
// A line wider than the terminal wraps where the terminal decides, and
// everything under it moves down a row — which on a form means the cursor
// is drawn somewhere other than where the reader is typing.
func TestNothingOnTheFormRunsOffTheWindow(t *testing.T) {
	for w := 60; w <= 160; w += 1 {
		s, e := atWidth(t, w, 40)

		rows := s.View(e.Frame.Body.H, e.Frame.Body.W, e)
		for i, line := range rows {
			if got := lipgloss.Width(line); got > e.Frame.Body.W {
				t.Fatalf("at %d columns: line %d is %d cells and the body is %d",
					w, i, got, e.Frame.Body.W)
			}
		}
	}
}

// TestTheFormSurvivesAWindowWithNoRoomInIt, which every window passes
// through while somebody drags its corner.
func TestTheFormSurvivesAWindowWithNoRoomInIt(t *testing.T) {
	s, e := form(t)

	for _, h := range []int{-1, 0, 1, 2} {
		rows := s.View(h, e.Frame.Body.W, e)

		if h <= 0 && rows != nil {
			t.Errorf("a body of %d rows drew %d lines", h, len(rows))
		}

		for i, line := range rows {
			if got := lipgloss.Width(line); got > e.Frame.Body.W {
				t.Errorf("a body of %d rows: line %d is %d cells wide", h, i, got)
			}
		}
	}
}

// TestABoxOnTheFormHasAStraightRightEdge.
//
// The three borders of a box are built from three different subtractions,
// and only one of them has to be wrong for the edge to step. A reader sees
// a form that looks broken and cannot say why.
//
// The edge and not the line: the top border carries the paste tab outside
// its own corner on purpose, so what has to line up is where the box closes
// rather than how long the row is.
func TestABoxOnTheFormHasAStraightRightEdge(t *testing.T) {
	for w := 80; w <= 140; w += 4 {
		s, e := atWidth(t, w, 40)

		var (
			at    = -1
			lines []string
		)

		for _, line := range s.View(e.Frame.Body.H, e.Frame.Body.W, e) {
			edge := closesAt(line)
			if edge < 0 {
				continue
			}

			lines = append(lines, ansi.Strip(line))

			if at < 0 {
				at = edge
				continue
			}

			if edge != at {
				t.Errorf("at %d columns the box closes at column %d on one line and %d on another:\n%s",
					w, at, edge, strings.Join(lines, "\n"))

				break
			}
		}
	}
}

// closesAt is the column the box's right-hand border sits in, and -1 for a
// line that is not part of one.
func closesAt(line string) int {
	bare := ansi.Strip(line)

	at := -1

	for i, r := range []rune(bare) {
		if r == '│' || r == '┐' || r == '┘' {
			at = lipgloss.Width(string([]rune(bare)[:i]))
		}
	}

	return at
}
