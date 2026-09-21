package prose

// How a block of text is set: the head over a section, the facts beside a
// subject, a paragraph the model wrote, and a row of figures.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/markdown"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// TestASectionHeadRulesOutToTheEdge. The rule does the work a box would do —
// it says where one block ends — without spending two more lines and four
// more corners on saying it.
func TestASectionHeadRulesOutToTheEdge(t *testing.T) {
	for _, w := range []int{40, 80, 140} {
		got := ansi.Strip(Section("changes", "+3 −1", w, false))
		if lipgloss.Width(got) > w {
			t.Errorf("a head at width %d is %d cells wide", w, lipgloss.Width(got))
		}

		if !strings.Contains(got, "─") {
			t.Errorf("the head at width %d has no rule on it: %q", w, got)
		}
	}

	// A head with no room for a rule is the head alone rather than a head
	// with a negative one.
	if got := ansi.Strip(Section("changes", "", 8, true)); strings.Contains(got, "─") {
		t.Errorf("a head with no room drew a rule anyway: %q", got)
	}
}

// TestASectionAndTheFiguresUnderItCloseInTheSameColumn. The rule and the
// boxes are two blocks of one pane, and the pane reads as one thing only
// while they end in the same place: the rule subtracted the gutter twice —
// once inside the head that already carries it — and stopped two cells
// short of every box under it, at every width.
func TestASectionAndTheFiguresUnderItCloseInTheSameColumn(t *testing.T) {
	stats := []Stat{
		{Label: "cost", Value: "$1.20", Role: theme.OK},
		{Label: "duration", Value: "40m", Role: theme.Accent},
	}

	for w := 60; w <= 200; w++ {
		head := lipgloss.Width(Section("changes", "", w, true))
		if head != w-len(Gutter) {
			t.Fatalf("at %d columns the head closes at %d, want one gutter from the edge", w, head)
		}

		// The boxes are whole cells split between them, so the last one
		// can land a cell or two short of the rule; what it must not do
		// is close past it.
		if got := lipgloss.Width(Strip(stats, w)[0]); got > head {
			t.Fatalf("at %d columns the figures close at %d, past the rule at %d", w, got, head)
		}
	}
}

// TestOnlyAClosedSectionSaysWhatItIsHolding. An open one has its detail
// under it, and a count above detail that shows the same thing is a line the
// reader has to check against another line.
func TestOnlyAClosedSectionSaysWhatItIsHolding(t *testing.T) {
	shut := ansi.Strip(Section("changes", "+3 −1", 80, false))
	if !strings.Contains(shut, "+3 −1") {
		t.Errorf("a closed section does not say what it holds: %q", shut)
	}

	if open := ansi.Strip(Section("changes", "+3 −1", 80, true)); strings.Contains(open, "+3 −1") {
		t.Errorf("an open section repeats the count above its own detail: %q", open)
	}
}

// TestTheLabelIsUpperCasedWhateverItArrivesAs, so that two blocks named in
// two languages still read as the same kind of thing.
func TestTheLabelIsUpperCasedWhateverItArrivesAs(t *testing.T) {
	if got := ansi.Strip(Section("cambios", "", 80, true)); !strings.Contains(got, "CAMBIOS") {
		t.Errorf("the head drew %q, want the label upper-cased", got)
	}
}

// TestMetaDropsWhatIsNotThere, so a caller can pass a value it does not
// always have without asking first.
func TestMetaDropsWhatIsNotThere(t *testing.T) {
	got := ansi.Strip(Meta("ACME-7", "", "orbit", "   ", "done"))
	if strings.Count(got, "·") != 2 {
		t.Errorf("Meta drew %q, want two middots between three facts", got)
	}

	if Meta() != "" || Meta("", " ") != "" {
		t.Errorf("Meta with nothing to say drew %q", Meta("", " "))
	}
}

// TestAQuotedParagraphIsRuledAndWrapped. It is what the model wrote, and the
// rule is what says so: a paragraph set at the pane's full width loses the
// reader at each line break.
func TestAQuotedParagraphIsRuledAndWrapped(t *testing.T) {
	text := strings.Repeat("the model wrote this and kept writing it. ", 12)

	lines := Quote(text, 80, Gutter)
	if len(lines) < 2 {
		t.Fatalf("a long paragraph came back as %d line(s), want it wrapped", len(lines))
	}

	for _, l := range lines {
		if !strings.Contains(ansi.Strip(l), "│") {
			t.Errorf("a quoted line has no rule beside it: %q", ansi.Strip(l))
		}

		if got := lipgloss.Width(l); got > 80 {
			t.Errorf("a quoted line is %d cells wide in a pane of 80: %q", got, ansi.Strip(l))
		}
	}

	// Blank paragraphs are the writer's spacing and not lines of their own
	// here: the rule would draw a bar beside nothing.
	if got := Quote("\n\n  \n", 80, Gutter); got != nil {
		t.Errorf("an empty paragraph drew %q", got)
	}
}

// TestAQuoteFillsThePaneAndStopsAtItsEdge. Wrapped prose lands wherever the
// last word before the measure ends, so a measure that is a few cells out
// never shows on prose — it shows on a token with nowhere to break, which
// is where a path or a hash in what the model wrote puts it. One long token
// per width says exactly where the measure is.
func TestAQuoteFillsThePaneAndStopsAtItsEdge(t *testing.T) {
	token := strings.Repeat("x", 400)

	for w := 30; w <= 160; w++ {
		var widest int

		for _, l := range Quote(token, w, Gutter) {
			widest = max(widest, lipgloss.Width(l))
		}

		// The indent and the rule stand to the left of the measure, and
		// one gutter is left to the right of it: the same column every
		// other block of the pane closes in, until the measure caps the
		// line at something narrower than the pane.
		want := min(w-len(Gutter), len(Gutter)+lipgloss.Width(rule)+markdown.Measure)
		if widest != want {
			t.Errorf("a quote in a pane of %d columns is %d cells wide, want %d", w, widest, want)
		}
	}
}

// TestTheStripIsBoxesWhenThereIsRoomAndOneLineWhenThereIsNot. Four figures
// side by side are read by comparing them, and a border is what tells the
// eye where one stops — but below sixty cells there is no room for four of
// anything.
func TestTheStripIsBoxesWhenThereIsRoomAndOneLineWhenThereIsNot(t *testing.T) {
	stats := []Stat{
		{Label: "cost", Value: "$1.20", Role: theme.OK},
		{Label: "duration", Value: "40m", Role: theme.Accent},
		{Label: "flow", Value: "quick", Role: theme.Accent},
		{Label: "changed", Value: "+3 −1", Role: theme.Live},
	}

	wide := Strip(stats, 140)
	if len(wide) < 3 {
		t.Fatalf("a wide strip drew %d lines, want the boxes", len(wide))
	}

	for _, l := range wide {
		if got := lipgloss.Width(l); got > 140 {
			t.Errorf("a row of the strip is %d cells wide in a pane of 140", got)
		}
	}

	narrow := Strip(stats, 50)
	if len(narrow) != 1 {
		t.Fatalf("a narrow strip drew %d lines, want the same figures on one", len(narrow))
	}

	for _, want := range []string{"$1.20", "40m", "quick", "+3 −1"} {
		if !strings.Contains(ansi.Strip(narrow[0]), want) {
			t.Errorf("the narrow strip dropped %q: %q", want, ansi.Strip(narrow[0]))
		}
	}

	if got := Strip(nil, 140); got != nil {
		t.Errorf("a strip of no figures drew %q", got)
	}

	// Sixty is where the boxes start, not where they stop: a pane exactly
	// that wide has the room the rule asks for.
	if got := Strip(stats, 60); len(got) < 3 {
		t.Errorf("a pane of 60 columns drew %d lines, want the boxes", len(got))
	}

	if got := Strip(stats, 59); len(got) != 1 {
		t.Errorf("a pane of 59 columns drew %d lines, want the one line", len(got))
	}
}
