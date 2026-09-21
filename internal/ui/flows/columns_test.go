package flows

// Where a pill is drawn is where a click on it lands.
//
// Every one of these walks the row that was drawn and asks the hit-test
// about the column the mark actually sits in, rather than about a column
// worked out the same way the hit-test works it out: two readings of one
// sum agree with each other however wrong they both are, which is how the
// pills on a flow's row came to stand three cells right of themselves.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/point"
)

// cellOf is the column a piece of text starts in, counted in cells: the
// terminal's own measure, and not the bytes or the runes before it.
func cellOf(row, sub string) int {
	before, _, found := strings.Cut(row, sub)
	if !found {
		return -1
	}

	return lipgloss.Width(before)
}

// TestTheMarkOnAPillIsWhereTheClickIs, for every pill this screen draws.
func TestTheMarkOnAPillIsWhereTheClickIs(t *testing.T) {
	e := world(t)
	s := Open(FromBoard, e)

	for at := range s.listed {
		s.sel = at

		row, line := -1, ""

		for i, l := range s.flowsListLines(e.Frame.Body.W, e) {
			if l.at == at && l.head {
				row, line = i, ansi.Strip(l.text)

				break
			}
		}

		if row < 0 {
			t.Fatalf("flow %d is on no row", at)
		}

		for _, c := range []struct{ mark, field string }{
			{"👁", "details"},
			{"✏", "edit"},
			{"🗑", "delete"},
		} {
			col := cellOf(line, c.mark)
			if col < 0 {
				continue // this flow does not offer that one
			}

			got := s.Hit(col, e.Frame.Body.Y+row, e)
			if got.Field != c.field {
				t.Errorf("%s: the %s mark is drawn in column %d and a click there is %q",
					s.listed[at].Name, c.field, col, got.Field)
			}
		}
	}
}

// TestTheMarksOnTheInstructionRowAreWhereTheClicksAre.
func TestTheMarksOnTheInstructionRowAreWhereTheClicksAre(t *testing.T) {
	s, e := designing(t, 130, 45)

	row, line := -1, ""

	lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)
	for i := start; i < len(lines); i++ {
		if lines[i].head && lines[i].field == flowFieldPrompt {
			row, line = i-start, ansi.Strip(lines[i].text)

			break
		}
	}

	if row < 0 {
		t.Fatal("the instructions row is not drawn")
	}

	for _, c := range []struct{ mark, field string }{
		{"📋", "paste_prompt"},
		{"✨", "autogen_prompt"},
		{"🗑", "clear_prompt"},
	} {
		col := cellOf(line, c.mark)
		if col < 0 {
			t.Errorf("the %s mark is not drawn: %q", c.field, line)

			continue
		}

		if got := s.Hit(col, e.Frame.Body.Y+row, e); got.Field != c.field {
			t.Errorf("the %s mark is in column %d and a click there is %q", c.field, col, got.Field)
		}
	}
}

// TestTheMarksOnTheButtonsRowAreWhereTheClicksAre.
func TestTheMarksOnTheButtonsRowAreWhereTheClicksAre(t *testing.T) {
	s, e := designing(t, 130, 45)

	row, line := -1, ""

	lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)
	for i := start; i < len(lines); i++ {
		if lines[i].field == flowFieldAddPhase {
			row, line = i-start, ansi.Strip(lines[i].text)

			break
		}
	}

	if row < 0 {
		t.Fatal("the buttons row is not drawn")
	}

	for _, c := range []struct{ mark, field string }{
		{"+", "add_phase"},
		{"🗑", "del_phase"},
		{"✔", "save"},
	} {
		col := cellOf(line, c.mark)
		if col < 0 {
			t.Errorf("the %s button is not drawn: %q", c.field, line)

			continue
		}

		if got := s.Hit(col, e.Frame.Body.Y+row, e); got.Field != c.field {
			t.Errorf("the %s button is in column %d and a click there is %q", c.field, col, got.Field)
		}
	}
}

// TestARowPastTheLastOneAnswersNothing, on the list and in the designer.
// The rows drawn are read from the same list the drawing uses, so the row
// after the last is the one to ask about: it is what a click on the blank
// under a short list lands on.
func TestARowPastTheLastOneAnswersNothing(t *testing.T) {
	e := world(t)

	s := Open(FromBoard, e)
	lines := s.flowsListLines(e.Frame.Body.W, e)

	for _, row := range []int{len(lines), len(lines) + 1, len(lines) + 40} {
		if got := s.Hit(4, e.Frame.Body.Y+row, e); got.Kind != point.None {
			t.Errorf("row %d of a list of %d answers %v %q", row, len(lines), got.Kind, got.Field)
		}
	}

	built, _ := Open(FromBoard, e).editFlow("careful", e)

	form, start := built.builderView(e.Frame.Body.H, e.Frame.Body.W, e)
	for _, row := range []int{len(form) - start, len(form) - start + 1, len(form) + 40} {
		if got := built.Hit(4, e.Frame.Body.Y+row, e); got.Kind != point.None {
			t.Errorf("row %d of a form of %d answers %v %q", row, len(form), got.Kind, got.Field)
		}
	}
}
