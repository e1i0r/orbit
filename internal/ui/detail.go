package ui

// The task view: one level below the board, three tabs behind one key.
//
// The board answers "what needs me". This screen answers "what happened",
// and it answers it three ways — the record, the changes, and what the
// engine actually printed. They are tabs rather than three panes side by
// side because at eighty columns three panes are three columns of twenty-six
// cells, and a diff in twenty-six cells is not a diff.
//
// The frame is four regions and they are computed in one place, paneHeight,
// so that what syncPanes sizes the viewport to and what detailRows draws
// cannot drift apart: a pane one row taller than the space it is drawn into
// is a frame one row too tall, which scrolls the whole terminal.

import (
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// tab is which of the three panes is on top. The order is the order they are
// read in: what happened, what changed, what was said.
type tab int

const (
	tabLog tab = iota
	tabDiff
	tabEvidence
	tabCount
)

// paneHeight is how many rows of a body of h the pane itself gets.
//
// Two rows always go to the heading and the tab strip: a screen that does
// not say which task it is open on, and which of three tabs is showing, is
// worse than a screen with two fewer rows of content. The advice line at the
// bottom is given up first, at three rows and under, because it is the only
// one of the four that says nothing a reader cannot find out by pressing a
// key.
func paneHeight(h int) int {
	if h >= 4 {
		return h - 3
	}
	return max(h-2, 0)
}

// detailRows draws the task view into the body region.
func (m Model) detailRows(h, w int) []string {
	if h <= 0 || w <= 0 {
		return nil
	}
	out := []string{m.detailHead(w)}
	if h > 1 {
		out = append(out, m.tabStrip(w))
	}
	if n := paneHeight(h); n > 0 {
		out = append(out, m.paneRows(n, w)...)
	}
	if h >= 4 {
		out = append(out, fit(m.moreLine(), w))
	}
	return fill(out, h)
}

// detailHead names the task the view is open on: its id and title on the
// left, the repository and what the row said about it on the right.
//
// It repeats the row the reader pressed ⏎ on, and that is the point. A pane
// with no heading is a pane a reader can leave open, look away from, and
// come back to believing it is about a different task — which in the program
// this replaces was how a diff got applied to the wrong branch.
func (m Model) detailHead(w int) string {
	t, ok := m.task(m.detail)
	left := Paint(Accent).Render(m.detail)
	if !ok {
		// The record outlives the board: a task filtered out, or finished
		// and rolled off, is still perfectly readable one level down. Saying
		// so is better than a heading that quietly loses its right-hand half
		// while the pane below it goes on showing history.
		return spread(" "+left, Paint(Dim).Render(m.opts.Words.T("detail.gone",
			"this task is no longer on the board")), w)
	}
	if t.Title != "" {
		left += "  " + Paint(Dim).Render(t.Title)
	}
	word, role := m.stateWord(t)
	right := Paint(Dim).Render(t.Repo)
	if word != "" {
		right += Paint(Dim).Render(dot) + Paint(role).Render(word)
	}
	return spread(" "+left, right, w)
}

// tabStrip is the three tabs, with the one showing marked.
//
// The mark is the same glyph the cursor uses on the board, for the same
// reason: it survives a terminal that has no colour at all, where a tab
// distinguished only by being brighter is a tab nobody can tell from the
// other two.
func (m Model) tabStrip(w int) string {
	p := m.opts.Words
	names := []struct {
		tab  tab
		text string
	}{
		{tabLog, p.T("tab.log", "log")},
		{tabDiff, p.T("tab.diff", "diff")},
		{tabEvidence, p.T("tab.evidence", "evidence")},
	}
	var parts []string
	for _, n := range names {
		if n.tab == m.tab {
			parts = append(parts, Paint(Accent).Render(markGlyph+n.text))
			continue
		}
		parts = append(parts, Paint(Dim).Render(" "+n.text))
	}
	var right string
	if a := m.attempt(); a > 0 {
		right = Paint(Dim).Render(p.T("log.attempt", "attempt {n}", about("n", strconv.Itoa(a))))
	}
	return spread(" "+strings.Join(parts, "  "), right, w)
}

// paneRows is the pane itself, cut to the region it was given.
//
// Every line is passed through fit even though the viewport already cut it
// to its own width. The two widths are the same number arrived at twice, and
// the assertion the measured render makes is about this frame rather than
// about the viewport's arithmetic: a diff line of four thousand cells must
// scroll inside the pane and must never widen the window around it.
func (m Model) paneRows(h, w int) []string {
	lines := strings.Split(m.panes[m.tab].View(), "\n")
	out := make([]string, 0, h)
	for _, line := range lines {
		out = append(out, fit(line, w))
	}
	return fill(out[:min(len(out), h)], h)
}

// moreLine is the "there is more" line, and it says something different
// depending on what the pane is doing.
//
// A pane whose content fits says nothing at all. Advice about scrolling on a
// screen with nothing below the fold is furniture, and furniture in the one
// row that is supposed to mean "there is more down there" is how that row
// stops being read.
//
// The brief asks for a third sentence — how to reach the pane at all, for a
// reader whose keystrokes are going somewhere else. Nothing in this window
// can be in that state: the two things that take the keyboard first, the
// filter prompt and a confirmation, are both raised from the board and
// neither can be raised from here, and this line is only ever drawn with the
// task view on top. It is left out rather than written unreachable, and the
// day a verb here raises a question is the day it comes back with it.
func (m Model) moreLine() string {
	vp := m.panes[m.tab]
	if vp.AtTop() && vp.AtBottom() {
		return ""
	}
	p := m.opts.Words
	if m.tab == tabLog && m.following {
		return " " + Paint(Live).Render(p.T("detail.following", "following — {key} stops it",
			about("key", m.keys.Up.Help().Key)))
	}
	return " " + Paint(Dim).Render(p.T("detail.scrolls", "{keys} scrolls",
		about("keys", m.keys.Up.Help().Key+m.keys.Down.Help().Key)))
}

// subject is the task the view is open on, taken from the board every frame
// rather than copied when the view opened.
//
// That is what makes the heading move: a task that finishes while its log is
// on screen changes its own row on the next refresh, and this screen has to
// say the same thing that row says. A task that has left the board entirely
// leaves the id standing on its own, which is the honest answer — the record
// is still readable, and the heading says which one it is.
func (m Model) subject() view.Task {
	if t, ok := m.task(m.detail); ok {
		return t
	}
	return view.Task{ID: m.detail}
}

// attempt is which run the newest entry belongs to, which is what the strip
// says on the right. It is 0 for a task nobody has run.
func (m Model) attempt() int {
	if len(m.entries) == 0 {
		return 0
	}
	return m.entries[len(m.entries)-1].Attempt
}

// detailHints is the key bar under the task view. It is a different list
// from the board's because every verb on the board acts on the row under the
// cursor, and there is no cursor here.
func (m Model) detailHints() []string {
	out := []string{hintFor(m.keys.Back), hintFor(m.keys.NextTab)}
	if m.tab == tabDiff {
		out = append(out, hintFor(m.keys.Edit))
	}
	return append(out, hint(m.keys.Up.Help().Key+m.keys.Down.Help().Key,
		m.opts.Words.T("key.scroll", "scroll")))
}

// syncPanes rebuilds all three panes and re-sizes them to the region.
//
// All three are rebuilt whenever any of the facts behind them changes, which
// costs a walk of a few dozen lines and buys the thing tabbing has to be:
// instant, and the same content it would have had if the reader had been
// looking at it all along. A pane built lazily on the first tab to it is a
// pane whose scroll position resets every time a message arrives.
func (m Model) syncPanes() Model {
	w, h := max(m.frame.Body.W, 1), max(paneHeight(m.frame.Body.H), 1)
	content := [tabCount][]string{
		tabLog:      m.logLines(),
		tabDiff:     m.diffLines(),
		tabEvidence: m.evidenceLines(),
	}
	for i := range m.panes {
		vp := m.panes[i]
		vp.SoftWrap = false
		vp.SetWidth(w)
		vp.SetHeight(h)
		vp.SetContentLines(content[i])
		m.panes[i] = vp
	}
	if m.following {
		vp := m.panes[tabLog]
		vp.GotoBottom()
		m.panes[tabLog] = vp
	}
	return m
}
