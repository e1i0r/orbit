//go:build integration

package cli

// The mouse, which the window promises reaches everything the keyboard does.
//
// The rule is written down in the cockpit's own skill — "mouse and keyboard
// both reach everything; a screen that can only be driven by one of the two
// is unfinished" — and until now nothing checked it.
//
// Every click here is aimed by looking for the thing in the frame first, so
// what is asserted is that what a reader can see, a reader can click, where
// they see it. None of these tests knows a coordinate.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// TestCockpitAClickOpensABandTheSameWayEnterDoes.
func TestCockpitAClickOpensABandTheSameWayEnterDoes(t *testing.T) {
	ctx, dir := cockpitBoard(t)

	m := opened(t, ctx, dir)

	byKey := drawn(t, press(t, m, named(tea.KeyEnter)))
	byMouse := drawn(t, clickOn(t, m, "TO DO"))

	if !strings.Contains(byMouse, "LED-1") {
		t.Fatalf("clicking the band's head did not open it:\n%s", byMouse)
	}

	if byMouse != byKey {
		t.Errorf("the mouse and the keyboard drew different boards:\n%s\n\nand:\n%s", byMouse, byKey)
	}
}

// TestCockpitAClickOpensATask, which is the gesture the whole board is for.
func TestCockpitAClickOpensATask(t *testing.T) {
	ctx, dir := cockpitBoard(t)

	m := opened(t, ctx, dir)
	board := drawn(t, m)

	open := clickOn(t, m, "TO DO")
	task := drawn(t, clickOn(t, open, "LED-1"))

	if task == board {
		t.Fatalf("clicking the row did not open the task:\n%s", task)
	}

	if !strings.Contains(task, "LED-1") {
		t.Errorf("what opened is not the task that was clicked:\n%s", task)
	}
}

// TestCockpitTheAutopilotChipAnswersItsOwnMouse. The bar draws the switch and
// the key that flips it; clicking the chip is the other way to the same
// thing, and the pip is what says which way it went.
func TestCockpitTheAutopilotChipAnswersItsOwnMouse(t *testing.T) {
	ctx, dir := cockpitBoard(t)

	m := opened(t, ctx, dir)

	before := drawn(t, m)
	if !strings.Contains(before, "autopilot ○") {
		t.Fatalf("the board does not open with autopilot off:\n%s", before)
	}

	after := drawn(t, clickOn(t, m, "autopilot"))
	if !strings.Contains(after, "autopilot ●") {
		t.Errorf("clicking the autopilot chip did not turn it on:\n%s", after)
	}
}

// TestCockpitEveryAffordanceTheBarDrawsIsClickable.
//
// The bar names its keys in brackets — [n], [S], [F], [m] — and each of them
// is a promise that the thing is there to be pressed. This walks whatever the
// bar happens to be drawing rather than a list written here, so an
// affordance added to the bar is covered the day it is added.
func TestCockpitEveryAffordanceTheBarDrawsIsClickable(t *testing.T) {
	ctx, dir := cockpitBoard(t)

	m := opened(t, ctx, dir)
	board := drawn(t, m)

	offered := bracketed(lines(t, m))
	if len(offered) < 3 {
		t.Fatalf("the bar offers %v, which is too few to be the bar:\n%s", offered, board)
	}

	for _, key := range offered {
		// [q] is deliberately not clickable — a window that closed on a
		// stray click in the corner loses whatever was being read. [c]
		// hands the terminal to another program, and [n] starts a run,
		// which is the run tests' subject and needs an engine.
		if key == "[q]" || key == "[c]" || key == "[n]" {
			continue
		}

		if got := drawn(t, clickOn(t, m, key)); got == board {
			t.Errorf("clicking %s drew the same board, so nothing answered it", key)
		}
	}
}

// bracketed is every [x] the bar is drawing, which is every affordance it is
// promising.
func bracketed(frame []string) []string {
	var found []string

	for _, line := range frame[max(len(frame)-3, 0):] {
		for i := 0; i < len(line); i++ {
			if line[i] != '[' {
				continue
			}

			if end := strings.IndexByte(line[i:], ']'); end > 1 && end <= 4 {
				found = append(found, line[i:i+end+1])
			}
		}
	}

	return found
}

// TestCockpitEachHintAnswersItsOwnCellsAndNobodyElsesDoes.
//
// This is the failure the window keeps having: a click that does nothing, or
// one that does what the thing beside it does. Both are the same fault — the
// zone a click is matched against and the text a reader sees have drifted
// apart — and neither is caught by clicking one cell and seeing something
// happen.
//
// So every cell of every hint is clicked, and two things are asked of the
// answers: that a hint answers the same way at both of its ends, and that no
// two hints answer the same way. The second is what "I pressed close and it
// opened" looks like from here.
func TestCockpitEachHintAnswersItsOwnCellsAndNobodyElsesDoes(t *testing.T) {
	ctx, dir := cockpitBoard(t)

	m := opened(t, ctx, dir)
	board := drawn(t, m)
	frame := lines(t, m)

	y := rowOf(t, frame, "[?]")
	answers := map[string]string{}

	for _, s := range spans(frame[y]) {
		if exempt(s.text) {
			continue
		}

		first, last := drawn(t, clicked(t, m, s.from, y)), drawn(t, clicked(t, m, s.to, y))
		if first != last {
			t.Errorf("%q (cells %d..%d) answers one thing at its first cell and another at its last:\nfirst: %s\nlast:  %s",
				s.text, s.from, s.to, tailOf(first), tailOf(last))
			continue
		}

		if first == board {
			// A hint that answers nothing at all is the other half of the
			// same failure: it is drawn, so it is promised.
			t.Errorf("%q is drawn on the bar and answers no click", s.text)
			continue
		}

		if was, seen := answers[first]; seen {
			t.Errorf("%q and %q answer the same thing — one of the two is over the other's cells",
				s.text, was)
		}

		answers[first] = s.text
	}

	if len(answers) < 3 {
		t.Fatalf("only %d hints answered anything, which is not the bar", len(answers))
	}
}

// exempt is the hints this test does not press, and why.
//
// [A] is the one that matters: what it changes is a saved setting and not
// anything in this window, so clicking it twice puts it back — a sweep that
// pressed every cell of it would read its own toggling as cells that do not
// answer. It is covered on its own, one click at a time, above.
func exempt(text string) bool {
	switch {
	case strings.HasPrefix(text, "[q]"): // closing the window is not a click's to make
		return true
	case strings.HasPrefix(text, "[c]"): // hands the terminal to another program
		return true
	case strings.HasPrefix(text, "[n]"): // starts a run, which wants an engine
		return true
	case strings.HasPrefix(text, "[A]"): // a saved setting: see above
		return true
	case strings.HasPrefix(text, "[↑↓]"): // a click cannot mean up and down
		return true
	}

	return false
}

// tailOf is the end of a frame, which is where the bar is.
func tailOf(frame string) string {
	if len(frame) < 200 {
		return frame
	}

	return "…" + frame[len(frame)-200:]
}

// span is one hint as it was drawn: the text, and the first and last cell it
// covers.
type span struct {
	text     string
	from, to int
}

// spans reads the hints off the bar as a reader sees them, in cells.
//
// In cells and not in bytes, which is the same mistake this test exists to
// catch: `[↑↓]` is four cells and eight bytes, and every column after it in a
// byte count names somewhere else. A gap is two spaces, or one space in front
// of a bracket — which is how the three in the corner are joined.
func spans(bar string) []span {
	var (
		out    []span
		text   string
		from   int
		column int
	)

	closeOne := func(to int) {
		if trimmed := strings.TrimSpace(text); strings.HasPrefix(trimmed, "[") {
			out = append(out, span{text: trimmed, from: from, to: to})
		}

		text = ""
	}

	runes := []rune(bar)

	for i := 0; i < len(runes); i++ {
		w := lipgloss.Width(string(runes[i]))

		if runes[i] == ' ' {
			twice := i+1 < len(runes) && runes[i+1] == ' '
			before := i+1 < len(runes) && runes[i+1] == '['

			if (twice || before) && text != "" {
				closeOne(column - 1)
			} else if text != "" {
				text += " "
			}

			column += w

			continue
		}

		if text == "" {
			from = column
		}

		text += string(runes[i])
		column += w
	}

	closeOne(column - 1)

	return out
}

// rowOf is the line something was drawn on.
func rowOf(t *testing.T, frame []string, needle string) int {
	t.Helper()

	for y, line := range frame {
		if strings.Contains(line, needle) {
			return y
		}
	}

	t.Fatalf("%q is nowhere in the frame", needle)

	return 0
}
