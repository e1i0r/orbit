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

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// paneHeight is how many rows of a body of h the pane itself gets.
//
// Two rows always go to the heading and the tab strip: a screen that does
// not say which task it is open on, and which of the tabs is showing, is
// worse than a screen with two fewer rows of content.
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

// tabStrip renders the eleven tabs.
//
// The tab strip at narrow columns cannot fit eleven names. It shows the
// key numbers (1-9, 0, w) with only the active pane's name spelled out, and
// the full catalogue is accessible via the help overlay [?].
// tabStrip renders the eleven tabs.
func (m Model) tabStrip(w int) string {
	p := m.opts.Words
	var fullParts []string
	for _, n := range m.tabNames() {
		k := paneKey(n.tab)
		tag := "[" + k + " " + n.text + "]"
		if n.tab == m.tab {
			fullParts = append(fullParts, Paint(Accent).Render(tag))
		} else {
			fullParts = append(fullParts, Paint(Dim).Render(tag))
		}
	}

	var parts []string
	if lipgloss.Width(strings.Join(fullParts, " "))+10 <= w {
		parts = fullParts
	} else {
		for _, n := range m.tabNames() {
			k := paneKey(n.tab)
			if n.tab == m.tab {
				parts = append(parts, Paint(Accent).Render("["+k+" "+n.text+"]"))
			} else {
				parts = append(parts, Paint(Dim).Render("["+k+"]"))
			}
		}
	}

	var right string
	switch attempt := m.attempt(); {
	case m.tab == tabDiff && m.diffKnown && m.diffErr == nil && m.diffNoBase:
		right = Paint(Dim).Render(p.T("diff.no_base", "no base branch"))
	case attempt > 0:
		right = Paint(Dim).Render(p.T("log.attempt", "attempt {n}", about("n", strconv.Itoa(attempt))))
	}
	return spread(" "+strings.Join(parts, " "), right, w)
}

// placedTab is one tab of the drawn strip and the cells it occupies.
type placedTab struct {
	tab  tab
	x, w int
}

// placeTabs walks the strip matching what tabStrip drew.
func (m Model) placeTabs() []placedTab {
	out := make([]placedTab, 0, tabCount)
	x := 1
	var fullWidth int
	for _, n := range m.tabNames() {
		fullWidth += lipgloss.Width("["+paneKey(n.tab)+" "+n.text+"]") + 1
	}
	useFull := fullWidth+10 <= m.frame.Body.W

	for _, n := range m.tabNames() {
		k := paneKey(n.tab)
		text := "[" + k + "]"
		if useFull || n.tab == m.tab {
			text = "[" + k + " " + n.text + "]"
		}
		cells := lipgloss.Width(text)
		out = append(out, placedTab{tab: n.tab, x: x, w: cells})
		x += cells + 1
	}
	return out
}

// paneRows is the pane itself, cut to the region it was given.
func (m Model) paneRows(h, w int) []string {
	lines := strings.Split(m.panes[m.tab].View(), "\n")
	out := make([]string, 0, h)
	for _, line := range lines {
		out = append(out, fit(line, w))
	}
	return fill(out[:min(len(out), h)], h)
}

// moreLine is the "there is more" scroll advice line.
func (m Model) moreLine() string {
	vp := m.panes[m.tab]
	if vp.AtTop() && vp.AtBottom() {
		return ""
	}
	p := m.opts.Words
	if m.tab == tabTimeline && m.following {
		return " " + Paint(Live).Render(p.T("detail.following", "following — {key} stops it",
			about("key", m.keys.Up.Help().Key)))
	}
	return " " + Paint(Dim).Render(p.T("detail.scrolls", "{keys} scrolls",
		about("keys", m.keys.Up.Help().Key+m.keys.Down.Help().Key)))
}

// subject is the task the view is open on.
func (m Model) subject() view.Task {
	if t, ok := m.task(m.detail); ok {
		return t
	}
	return view.Task{ID: m.detail}
}

// attempt is which run the newest entry belongs to.
func (m Model) attempt() int {
	if len(m.entries) == 0 {
		return 0
	}
	return m.entries[len(m.entries)-1].Attempt
}

// detailHints is the key bar under the task view.
func (m Model) detailHints() []barHint {
	out := []barHint{hintFor(m.keys.Back), hintFor(m.keys.NextTab)}
	if m.tab == tabDiff {
		out = append(out, hintFor(m.keys.Edit))
	}
	return append(out, hint(m.keys.Up.Help().Key+m.keys.Down.Help().Key,
		m.opts.Words.T("key.scroll", "scroll")))
}
