package supervisor

// The column of what Orbit knows, down the right of this screen.
//
// It takes its width from the thread, so where it starts and how much it
// gives back are one sum read from both sides — and it stops listing when
// the window is short, which is the one place a reader can be told they
// have seen everything Orbit knows when they have not.

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// rule is one fact for the column beside the thread.
func rule(id, phrase string, at time.Time) knowledge.Rule {
	return knowledge.Rule{
		ID: id, Scope: knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human, Phrase: phrase, At: at,
	}
}

// TestTheSideIsDrawnFromTheWidthItNeedsAndNotACellLess. Seventy columns for
// the thread, three of gap and thirty of its own: at that width the column
// appears, and one cell under it the screen is the thread alone.
func TestTheSideIsDrawnFromTheWidthItNeedsAndNotACellLess(t *testing.T) {
	s, _ := opened(t, saidThree())
	s.knows = []knowledge.Rule{rule("aaaa1111", "never log a card number", time.Time{})}

	// The gutter the screen is indented by, the cell the rail stands in,
	// the gap, the column, and a thread still worth reading.
	const fits = sideMinThread + gutter + 1 + sideGap + sideWidth

	for _, c := range []struct {
		w    int
		want bool
	}{
		{fits - 1, false},
		{fits, true},
		{fits + 40, true},
	} {
		if got := s.sideFits(c.w); got != c.want {
			t.Errorf("a body of %d columns fits the side = %v, want %v", c.w, got, c.want)
		}
	}

	// And a screen with nothing to say draws no column at whatever width.
	bare := s
	bare.knows = nil

	if bare.sideFits(fits + 40) {
		t.Error("a screen that knows nothing drew the column anyway")
	}
}

// TestTheSideStartsInTheSameColumnOnEveryRow. It is put beside rows wrapped
// to the thread's width plus the cell the scroll rail stands in, so a seam
// computed from the text alone would take the rail off and leave an
// ellipsis down the side of every row that had one.
func TestTheSideStartsInTheSameColumnOnEveryRow(t *testing.T) {
	e := wideWorld(t, saidThree())
	e.Knows = func() []knowledge.Rule {
		return []knowledge.Rule{rule("aaaa1111", "never log a card number", time.Time{})}
	}

	s := Open(0, e)

	body := e.Frame.Body

	cw, _ := s.layout(body.H, body.W, e)

	rows := s.View(body.H, body.W, e)

	found := false

	for _, r := range rows {
		line := ansi.Strip(r)

		at := strings.Index(line, "Brain")
		if at < 0 {
			continue
		}

		found = true

		// The gutter the whole screen is indented by, the thread and the
		// cell its rail stands in, and the gap between the two columns.
		if want := gutter + cw + 1 + sideGap; at != want {
			t.Errorf("the side starts at column %d, want %d", at, want)
		}
	}

	if !found {
		t.Fatal("the column beside the thread is not drawn at all")
	}

	// And the column is drawn whole: what is cut off a sentence about a
	// rule is the half that says what the rule actually is.
	phrase := "never log a card number"

	said := false

	for _, r := range rows {
		if strings.Contains(ansi.Strip(r), phrase) {
			said = true
		}
	}

	if !said {
		t.Errorf("the column does not say %q whole:\n%s", phrase, ansi.Strip(strings.Join(rows, "\n")))
	}
}

// TestAFactOnTheSideIsWrappedInsideIt. The column is thirty cells and the
// sentence in it is indented by two, so a wrap against the column itself
// puts the tail of every sentence past the edge of the window.
func TestAFactOnTheSideIsWrappedInsideIt(t *testing.T) {
	s, e := opened(t, saidThree())

	long := rule("aaaa1111",
		"never log a card number, not the last four either, and not in a stack trace", time.Time{})

	for _, r := range s.sideFact(long, e) {
		if got := lipgloss.Width(r); got > sideWidth {
			t.Errorf("a row of the column is %d cells wide in a column of %d: %q",
				got, sideWidth, ansi.Strip(r))
		}
	}
}

// TestAColumnThatRunsOutOfRoomSaysHowMuchIsLeft. A column that quietly
// stops listing is worse than a short one: whoever reads it believes they
// have seen everything Orbit knows, and the facts past the bottom are
// exactly the ones nobody finds out about.
func TestAColumnThatRunsOutOfRoomSaysHowMuchIsLeft(t *testing.T) {
	s, e := opened(t, saidThree())

	rows := []string{"Brain", "", "general", "  the first fact", "general", "  the second fact"}

	// Room for all of it: nothing is said about more, because there is no
	// more.
	whole := s.cutSide(rows, len(rows), 2, e)
	if len(whole) != len(rows) {
		t.Fatalf("a column of %d rows in %d came back as %d", len(rows), len(rows), len(whole))
	}

	for _, r := range whole {
		if strings.Contains(ansi.Strip(r), "more") {
			t.Errorf("a column with room for everything says %q", ansi.Strip(r))
		}
	}

	// One row short: the last row becomes the count, and it counts the
	// facts that are not on the list rather than the rows.
	cut := s.cutSide(rows, len(rows)-1, 2, e)
	if len(cut) != len(rows)-1 {
		t.Fatalf("a column of %d rows cut to %d came back as %d", len(rows), len(rows)-1, len(cut))
	}

	last := ansi.Strip(cut[len(cut)-1])
	if !strings.Contains(last, "1") || !strings.Contains(last, "more") {
		t.Errorf("the last row of a cut column is %q, want one more", last)
	}
}

// TestOnlyAFactWrittenWhileTheReaderWasAwayIsMarked, and only while it is
// still what they missed. A mark that never goes away stops being news.
func TestOnlyAFactWrittenWhileTheReaderWasAwayIsMarked(t *testing.T) {
	s, e := opened(t, saidThree())

	for _, c := range []struct {
		name string
		f    knowledge.Rule
		want bool
	}{
		{"just written by a run", knowledge.Rule{
			Source: knowledge.FromRecord, At: e.Now.Add(-time.Minute),
		}, true},
		{"a minute inside the day", knowledge.Rule{
			Source: knowledge.FromRecord, At: e.Now.Add(-sinceLearned + time.Minute),
		}, true},
		{"exactly a day old", knowledge.Rule{
			Source: knowledge.FromRecord, At: e.Now.Add(-sinceLearned),
		}, false},
		{"older than a day", knowledge.Rule{
			Source: knowledge.FromRecord, At: e.Now.Add(-sinceLearned - time.Minute),
		}, false},
		{"typed by the reader", knowledge.Rule{
			Source: knowledge.Human, At: e.Now.Add(-time.Minute),
		}, false},
		{"written down at no time at all", knowledge.Rule{
			Source: knowledge.FromRecord,
		}, false},
	} {
		if got := s.learnedRecently(c.f, e); got != c.want {
			t.Errorf("a fact %s is marked = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestASideTallerThanTheThreadKeepsBothOfThem. The two columns are drawn
// from two different counts, and a seam that walked the shorter of them
// would drop whichever column had more rows in it.
func TestASideTallerThanTheThreadKeepsBothOfThem(t *testing.T) {
	for _, c := range []struct {
		rows, side []string
	}{
		{[]string{"one", "two", "three"}, []string{"a", "b"}},
		{[]string{"one"}, []string{"a", "b", "c"}},
		{nil, []string{"a", "b"}},
	} {
		out := besideThread(c.rows, c.side, 20)

		if want := max(len(c.rows), len(c.side)); len(out) != want {
			t.Fatalf("%d rows beside %d side rows came back as %d, want %d",
				len(c.rows), len(c.side), len(out), want)
		}

		joined := ansi.Strip(strings.Join(out, "\n"))

		for _, text := range append(append([]string{}, c.rows...), c.side...) {
			if !strings.Contains(joined, text) {
				t.Errorf("%q is not on the screen:\n%s", text, joined)
			}
		}
	}
}
