package supervisor

// The thread: how wide it is drawn, which row of it the window starts at,
// and which message a row belongs to.

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/view"
)

// TestHowWideTheThreadIsDrawn. Every block of this screen is wrapped to one
// number, and it is three answers in one: the window less the gutter either
// side, capped so a sentence does not stretch across a monitor, floored so
// a narrow terminal still draws something — and cut again when the column
// beside it is up.
func TestHowWideTheThreadIsDrawn(t *testing.T) {
	s, e := opened(t, saidThree())

	for _, c := range []struct {
		w, want int
	}{
		{20, 24},  // under the floor
		{28, 24},  // at the floor
		{60, 56},  // the window less a gutter either side
		{100, 96}, //
		{114, 110},
		{200, 110}, // capped: a sentence does not stretch across a monitor
	} {
		if got, _ := s.layout(30, c.w, e); got != c.want {
			t.Errorf("a window of %d columns wraps the thread at %d, want %d", c.w, got, c.want)
		}
	}

	// With something to say down the side, the thread gives up what the
	// column needs and not a cell more.
	s.knows = []knowledge.Rule{rule("aaaa1111", "never log a card number", time.Time{})}

	const narrowest = sideMinThread + gutter + 1 + sideGap + sideWidth

	for _, c := range []struct {
		w, want int
	}{
		{narrowest - 1, narrowest - 1 - 4}, // too narrow for the side: the whole width
		{narrowest, sideMinThread},
		{narrowest + 10, sideMinThread + 10},
		{200, 110},
	} {
		if got, _ := s.layout(30, c.w, e); got != c.want {
			t.Errorf("with the side up, a window of %d wraps the thread at %d, want %d",
				c.w, got, c.want)
		}
	}
}

// TestAThreadShorterThanTheBodySitsAgainstTheFloor, and one exactly as tall
// as it is not padded at all. The last thing said sits against the line you
// answer it in: padding underneath left a conversation of two lines
// stranded at the top of a tall terminal.
func TestAThreadShorterThanTheBodySitsAgainstTheFloor(t *testing.T) {
	s, e := opened(t, saidThree())

	cw, _ := s.layout(30, e.Frame.Body.W, e)

	rendered, _ := s.threadLines(cw, e)
	if len(rendered) < 3 {
		t.Fatalf("three messages rendered as %d rows", len(rendered))
	}

	for _, c := range []struct {
		name    string
		maxRows int
		blanks  int
	}{
		{"a body with room to spare", len(rendered) + 4, 4},
		{"a body one row taller", len(rendered) + 1, 1},
		{"a body exactly as tall", len(rendered), 0},
	} {
		rows := s.supervisorBody(c.maxRows, cw, e)
		if len(rows) != c.maxRows {
			t.Fatalf("%s drew %d rows in %d", c.name, len(rows), c.maxRows)
		}

		blanks := 0

		for _, r := range rows {
			if ansi.Strip(r) != "" {
				break
			}

			blanks++
		}

		if blanks != c.blanks {
			t.Errorf("%s put %d blank rows above the thread, want %d", c.name, blanks, c.blanks)
		}

		// Whatever the padding, the last thing said is the last row.
		if got := ansi.Strip(rows[len(rows)-1]); got != ansi.Strip(rendered[len(rendered)-1]) {
			t.Errorf("%s ends on %q, want the last row of the thread", c.name, got)
		}
	}
}

// TestAThreadTallerThanTheBodyEndsWhereItIsBeingRead. Following is the
// bottom of the thread; picking is wherever the line being picked is, which
// the reader did not ask to scroll to and must not have to.
func TestAThreadTallerThanTheBodyEndsWhereItIsBeingRead(t *testing.T) {
	s, e := opened(t, &held{lines: longThreadLines(30)})

	cw, _ := s.layout(30, e.Frame.Body.W, e)

	rendered, starts := s.threadLines(cw, e)

	const maxRows = 8

	if len(rendered) <= maxRows {
		t.Fatalf("thirty messages rendered as %d rows", len(rendered))
	}

	// Following: the window ends on the last row there is.
	follow := s
	follow.follow = true

	if got := follow.threadOffset(len(rendered), maxRows, starts); got != len(rendered)-maxRows {
		t.Errorf("following left the window at row %d, want %d", got, len(rendered)-maxRows)
	}

	// Picking: whichever line is picked is inside the window, from either
	// end of the thread and from a pick that is no longer in it.
	pick := s
	pick.picking, pick.follow = true, false

	// From the top of the thread and from the bottom of it: a line above
	// the window is reached by scrolling up to its start, and one below it
	// by scrolling down to its end, and those are two different sums.
	for _, was := range []int{0, len(rendered) - maxRows} {
		for i := range len(starts) + 4 {
			pick.pick, pick.offset = i, was

			off := pick.threadOffset(len(rendered), maxRows, starts)
			if off < 0 || off > len(rendered)-maxRows {
				t.Fatalf("a pick of %d from row %d left the window at row %d of %d",
					i, was, off, len(rendered))
			}

			if i >= len(starts) {
				continue
			}

			// The whole of the message, not only the row it starts on:
			// a window that holds its first line and none of the rest
			// is one the reader cannot read what they are taking back in.
			end := len(rendered)
			if i+1 < len(starts) {
				end = starts[i+1]
			}

			if starts[i] < off || min(end, off+maxRows) <= starts[i] {
				t.Errorf("a pick of %d from row %d starts on row %d, outside the %d rows from %d",
					i, was, starts[i], maxRows, off)
			}
		}
	}
}

// TestOneMessageIsTheOneBeingPicked. The rail beside a message is what says
// which one a keystroke acts on, and this is the gesture that cannot be
// undone.
func TestOneMessageIsTheOneBeingPicked(t *testing.T) {
	s, e := opened(t, saidThree())

	s.picking = true

	cw, _ := s.layout(30, e.Frame.Body.W, e)

	for i := range s.lines {
		s.pick = i

		marked := 0

		for _, l := range s.lines {
			for _, r := range s.messageLines(l, cw, false, e) {
				if strings.Contains(ansi.Strip(r), "▶") {
					marked++
				}
			}
		}

		if marked != 0 {
			t.Fatalf("a message drawn unpicked carries the mark %d times", marked)
		}

		rows, starts := s.threadLines(cw, e)

		// The mark stands beside every row of the message it is on, and
		// beside no row of any other.
		end := len(rows)
		if i+1 < len(starts) {
			end = starts[i+1]
		}

		marked = 0

		for row, r := range rows {
			if !strings.Contains(ansi.Strip(r), "▶") {
				continue
			}

			marked++

			if row < starts[i] || row >= end {
				t.Errorf("with %d picked, a mark is on row %d, outside the rows %d to %d",
					i, row, starts[i], end)
			}
		}

		if marked == 0 {
			t.Errorf("with %d picked, no row carries the mark", i)
		}
	}
}

// TestAMessageAboutATaskSaysWhichOne, and one about nothing in particular
// says nothing: an empty pair of brackets beside a name reads as a task id
// that failed to load.
func TestAMessageAboutATaskSaysWhichOne(t *testing.T) {
	s, e := opened(t, saidThree())

	cw, _ := s.layout(30, e.Frame.Body.W, e)

	about := view.SupervisorLine{
		At: fixtureNow, Conversation: "c1", By: "operator", Channel: "tui",
		TaskID: "ACME-7", Text: "this one is stuck",
	}

	head := ansi.Strip(s.messageLines(about, cw, false, e)[0])
	if !strings.Contains(head, "ACME-7") {
		t.Errorf("a message about a task does not name it: %q", head)
	}

	plain := about
	plain.TaskID = ""

	if got := ansi.Strip(s.messageLines(plain, cw, false, e)[0]); strings.Contains(got, "()") {
		t.Errorf("a message about no task in particular draws %q", got)
	}
}

// TestWhatAMessageSaysIsWrappedInsideTheThread. The rail stands to the left
// of what was said, so a body wrapped against the thread's whole width puts
// the tail of every paragraph under the rail — where the next message is.
func TestWhatAMessageSaysIsWrappedInsideTheThread(t *testing.T) {
	s, e := opened(t, saidThree())

	// A word with nowhere to break says where the measure is; wrapped prose
	// lands wherever the last word before it ended, which is never the
	// measure itself.
	said := strings.Repeat("una frase que sigue y sigue ", 4) +
		strings.Repeat("palabralarguisima", 12)

	// The three ways a message is set: what a run wrote, read as Markdown;
	// what a person typed, which is a sentence; and one taken back.
	for _, l := range []view.SupervisorLine{
		{At: fixtureNow, Conversation: "c1", By: "zeta", Channel: "tui", Text: said},
		{At: fixtureNow, Conversation: "c1", By: "operator", Channel: "tui", Text: said},
		{At: fixtureNow, Conversation: "c1", By: "operator", Channel: "tui", Text: said, Retracted: true},
	} {
		for cw := 24; cw <= 120; cw++ {
			for _, r := range s.messageLines(l, cw, false, e) {
				if got := lipgloss.Width(r); got > cw {
					t.Errorf("a row of a message by %s is %d cells wide in a thread of %d",
						l.By, got, cw)
				}
			}
		}
	}
}

// longThreadLines is a conversation of n messages, all in one thread.
func longThreadLines(n int) []view.SupervisorLine {
	out := make([]view.SupervisorLine, 0, n)

	for i := range n {
		who := "operator"
		if i%2 == 1 {
			who = "zeta"
		}

		out = append(out, spoke(fixtureNow.Add(-time.Duration(n-i)*time.Minute), "c1", who,
			"a message of the thread, number "+strings.Repeat("x", i%7)))
	}

	return out
}
