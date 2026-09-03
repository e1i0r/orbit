package ui

// The task story on the overview: how this prompt became this diff, drawn as
// the chain it is.

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// storyLines is the five fields as a chain, or nothing at all.
//
// Nothing at all is the important half. Every task recorded before the story
// existed has none, and so has every flow whose last phase did not write
// one; a heading over five blank rows would be worse than the pane that was
// there before it, because it would look like a story that says nothing
// rather than a task that has none.
//
// A chain and not a list. Each field is the reason for the one under it —
// the route exists for the purpose, the symptom happened in it, the cause is
// under the symptom, the fix answers the cause — and indenting is the only
// way to say that in eighty columns without a diagram.
func (m Model) storyLines(w int) []string {
	story := m.newestStory()
	if story == nil {
		return nil
	}

	p := m.opts.Words

	out := []string{
		paneGutter + Text(Secondary).Render(p.T("overview.story", "how it happened")),
		"",
	}

	for i, step := range []struct {
		text  string
		about string
	}{
		{story.Entry, p.T("story.entry", "the way in")},
		{story.Purpose, p.T("story.purpose", "what it is for")},
		{story.Symptom, p.T("story.symptom", "what went wrong")},
		{story.Cause, p.T("story.cause", "why")},
		{story.Fix, p.T("story.fix", "what was done")},
	} {
		out = append(out, storyRow(i, step.text, step.about, w))
	}

	return append(out, m.walkLines(w)...)
}

// walkLines is what the task changed, under the story that says why.
//
// The claim and its evidence one line apart, which is the rule the whole
// story is built on: the model says what was done and the record says what
// was touched to do it, and a reader who doubts the first is one keystroke
// from the second.
//
// Every changed file, however many there are. What is pruned is the files
// the agent opened and left alone; a hundred changes draw a hundred rows,
// because a story that stopped at ten would be hiding the work rather than
// the noise.
func (m Model) walkLines(w int) []string {
	steps := view.Walk(m.entries)
	if len(steps) == 0 {
		return []string{""}
	}

	p := m.opts.Words

	out := []string{
		"",
		paneGutter + "        " + Text(Tertiary).Render(p.P("story.changed", len(steps),
			"{n} file changed, in the order it got there",
			"{n} files changed, in the order it got there")),
	}

	for _, s := range steps {
		out = append(out, paneGutter+"        "+Paint(OK).Render(fit(s.Path, max(20, w-2*len(paneGutter)-10)))+
			Text(Tertiary).Render(touches(p, s)))
	}

	return append(out, "")
}

// touches is how much was done to one file, said only when it is more than
// once: "changed" beside every row is a column of the same word.
func touches(p *words.Printer, s view.Step) string {
	if s.Touches < 2 {
		return ""
	}

	return "  " + p.P("story.touches", s.Touches, "{n} change", "{n} changes")
}

// storyRow is one link of the chain: the claim, and the word for which link
// it is.
//
// The label is dim and to the right because the claim is what a reader is
// here for; the label is what tells them, once, which of the five they are
// looking at.
func storyRow(depth int, text, about string, w int) string {
	branch := strings.Repeat("  ", depth)
	if depth > 0 {
		branch += "└─ "
	}

	line := branch + text

	// The label only when there is room for it and a gap between the two.
	// A label wrapped onto its own line reads as a sixth field.
	if room := w - lipgloss.Width(line) - lipgloss.Width(about) - 4; room > 0 {
		line += strings.Repeat(" ", room) + Text(Tertiary).Render(about)
	}

	return paneGutter + line
}

// newestStory is the story of the attempt that stands.
//
// The last one written and not the first: a task run three times told its
// story three times, and the two before it are about work that was thrown
// away.
func (m Model) newestStory() *view.Story {
	var found *view.Story

	for _, e := range m.entries {
		if e.Story != nil {
			found = e.Story
		}
	}

	return found
}
