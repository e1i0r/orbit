package flows

// One flow read on its own: what is on the screen at every height it is
// drawn at, and where a click on it lands.
//
// The reading is a diagram, a card per phase and a footer of three buttons.
// It was drawn whole into a body that then cut it, so on anything under
// about thirty rows a reader saw the first phase and nothing else — not the
// rest of them, not the buttons, and not the line saying which keys do what.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// read is the preview of one flow in a window of the given size.
func read(t *testing.T, name string, w, h int) (State, Env) {
	t.Helper()

	e := world(t)

	frame, err := layout.Fit(w, h)
	if err != nil {
		t.Fatalf("a window of %dx%d will not fit: %v", w, h, err)
	}

	e.Frame = frame

	return Preview(name, FromBoard, e), e
}

// seen is the screen as one string.
func seen(s State, e Env) string {
	return ansi.Strip(strings.Join(s.View(e.Frame.Body.H, e.Frame.Body.W, e), "\n"))
}

// TestTheWholeOfAFlowCanBeRead. Every phase of it is reachable at every
// height, and the ways out are on the screen the whole way down.
func TestTheWholeOfAFlowCanBeRead(t *testing.T) {
	for _, h := range []int{14, 18, 24, 40} {
		s, e := read(t, "careful", 100, h)

		// The footer never leaves: it is what a reader who has read to
		// the end reaches for.
		for _, want := range []string{"Select & Return", "Back"} {
			if !strings.Contains(seen(s, e), want) {
				t.Errorf("at %d rows the reading does not offer %q:\n%s", h, want, seen(s, e))
			}
		}

		// And every phase is on one of the screens a reader walking down
		// passes through — a row at a time, which is how they walk.
		var whole strings.Builder

		walked := s
		for range 60 {
			whole.WriteString(seen(walked, e) + "\n")

			walked = walked.Scroll(1)
		}

		for _, phase := range []string{"implement", "review", "fix"} {
			if !strings.Contains(whole.String(), phase) {
				t.Errorf("at %d rows, phase %q is on no screen the reader can reach:\n%s",
					h, phase, seen(s, e))
			}
		}

		// Walking back up reaches the head again.
		back := walked
		for range 90 {
			back = back.Scroll(-1)
		}

		if !strings.Contains(seen(back, e), "Workflow") {
			t.Errorf("at %d rows, walking back up does not reach the head:\n%s", h, seen(back, e))
		}
	}
}

// TestTheKeysThatMoveTheReading, which a reading this long needs and did not
// have: it answered three keys, and none of them moved it.
func TestTheKeysThatMoveTheReading(t *testing.T) {
	s, e := read(t, "careful", 100, 18)

	for _, c := range []struct {
		name string
		msg  tea.KeyPressMsg
		want int
	}{
		{"down", tea.KeyPressMsg{Code: tea.KeyDown}, 1},
		{"j", tea.KeyPressMsg{Code: 'j', Text: "j"}, 1},
		{"page down", tea.KeyPressMsg{Code: tea.KeyPgDown}, detailPage},
		{"end", tea.KeyPressMsg{Code: tea.KeyEnd}, detailEnd},
	} {
		next, out := s.Key(c.msg, e)
		if next.scroll != c.want {
			t.Errorf("%s from the top left the reading at row %d, want %d", c.name, next.scroll, c.want)
		}

		if out.Leave || next.Listing() {
			t.Errorf("%s left the reading", c.name)
		}
	}

	// And the other way, from wherever it was.
	down, _ := s.Key(tea.KeyPressMsg{Code: tea.KeyEnd}, e)

	for _, c := range []struct {
		name string
		msg  tea.KeyPressMsg
	}{
		{"up", tea.KeyPressMsg{Code: tea.KeyUp}},
		{"k", tea.KeyPressMsg{Code: 'k', Text: "k"}},
		{"page up", tea.KeyPressMsg{Code: tea.KeyPgUp}},
		{"home", tea.KeyPressMsg{Code: tea.KeyHome}},
	} {
		if next, _ := down.Key(c.msg, e); next.scroll >= down.scroll {
			t.Errorf("%s did not move the reading back: %d", c.name, next.scroll)
		}
	}

	// The top is the top: a reading walked up from it stays there.
	if next, _ := s.Key(tea.KeyPressMsg{Code: tea.KeyUp}, e); next.scroll != 0 {
		t.Errorf("up from the top left the reading at row %d", next.scroll)
	}
}

// TestEveryButtonIsWhereTheClickIs. The three pills are the only things on
// this screen to point at, and the hit-test had them by two column numbers
// written against their English labels, on a row found by counting back
// from a body that is padded to its own height — so it answered on the
// blank floor under the reading and never on a button.
func TestEveryButtonIsWhereTheClickIs(t *testing.T) {
	s, e := read(t, "careful", 100, 40)

	rows := s.View(e.Frame.Body.H, e.Frame.Body.W, e)

	// The row the buttons are drawn on, found by reading the screen.
	drawn := -1

	for i, r := range rows {
		if strings.Contains(ansi.Strip(r), "Select & Return") {
			drawn = i
		}
	}

	if drawn < 0 {
		t.Fatalf("the buttons are not drawn at all:\n%s", seen(s, e))
	}

	// Every column of every button answers, and answers with its own.
	at := 2

	for _, b := range detailButtons(e) {
		for _, x := range []int{at, at + lipgloss.Width(b.pill) - 1} {
			got := s.Hit(x, e.Frame.Body.Y+drawn, e)
			if got.Kind != point.FlowItem || got.Field != b.field {
				t.Errorf("column %d of the buttons row is %v %q, want %q", x, got.Kind, got.Field, b.field)
			}
		}

		// The gap between two buttons is not either of them.
		gap := at + lipgloss.Width(b.pill)
		if got := s.Hit(gap, e.Frame.Body.Y+drawn, e); got.Kind != point.None {
			t.Errorf("the gap at column %d answers %q", gap, got.Field)
		}

		at = gap + detailButtonGap
	}

	// The edit button carries which flow it would open.
	edit := s.Hit(2+lipgloss.Width(detailButtons(e)[0].pill)+detailButtonGap, e.Frame.Body.Y+drawn, e)
	if edit.ID != "careful" {
		t.Errorf("the edit button names flow %q, want careful", edit.ID)
	}

	// And no row above the buttons answers: the reading is a reading.
	for i := range drawn {
		if got := s.Hit(4, e.Frame.Body.Y+i, e); got.Kind != point.None {
			t.Errorf("row %d of the reading answers %v %q", i, got.Kind, got.Field)
		}
	}
}

// TestTheButtonsStayOnTheFloorWhereverTheReadingIs, which is what makes one
// coordinate enough to find them.
func TestTheButtonsStayOnTheFloorWhereverTheReadingIs(t *testing.T) {
	for _, h := range []int{14, 20, 30, 45} {
		s, e := read(t, "careful", 100, h)

		for _, at := range []State{s, s.Scroll(5), s.Scroll(detailEnd)} {
			rows := at.View(e.Frame.Body.H, e.Frame.Body.W, e)

			if len(rows) != e.Frame.Body.H {
				t.Fatalf("a body of %d rows drew %d", e.Frame.Body.H, len(rows))
			}

			foot, buttons := at.detailFoot(e.Frame.Body.H, e.Frame.Body.W, e)

			row := ansi.Strip(rows[e.Frame.Body.H-len(foot)+buttons])
			if !strings.Contains(row, "Select & Return") {
				t.Errorf("at %d rows and offset %d, the floor's buttons row is %q",
					h, at.scroll, row)
			}
		}
	}
}

// TestNoRowOfTheReadingRunsPastTheWindow, at every width and height it is
// drawn at.
func TestNoRowOfTheReadingRunsPastTheWindow(t *testing.T) {
	for w := layout.MinWidth; w <= 160; w += 7 {
		for _, h := range []int{14, 22, 40} {
			s, e := read(t, "careful", w, h)

			for _, at := range []State{s, s.Scroll(detailEnd)} {
				rows := at.View(e.Frame.Body.H, e.Frame.Body.W, e)
				for i, r := range rows {
					if got := lipgloss.Width(r); got > e.Frame.Body.W {
						t.Fatalf("at %dx%d row %d is %d cells wide in a body of %d",
							w, h, i, got, e.Frame.Body.W)
					}
				}
			}
		}
	}
}
