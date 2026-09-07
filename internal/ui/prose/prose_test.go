package prose

// How a block of text is set: the head over a section, the facts beside a
// subject, a paragraph the model wrote, and a row of figures.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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
}
