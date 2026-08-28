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

	head := m.detailHeadLines(w)

	out := append([]string{}, head...)
	if h >= len(head)+3 {
		out = append(out, "")
	}

	tabLine := len(out)
	if h > tabLine {
		out = append(out, m.tabStrip(w))
	}

	if m.tab == tabDiff && m.diffKnown && m.diffErr == nil {
		raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")

		files := parseDiffFiles(raw)
		if len(files) > 0 {
			activeIdx := fileIndexAtOffset(files, m.panes[tabDiff].YOffset())

			cursor := activeIdx
			if m.diffFilePicker {
				cursor = m.diffFileCursor
			}

			combo := renderDiffFileSelect(files, activeIdx, w, m.opts.Words, m.collapsedFiles, m.diffFilePicker, cursor)
			if combo != "" {
				parts := strings.Split(combo, "\n")
				out = append(out, parts...)
			}
		}
	}

	topUsed := len(out)
	if n := max(0, h-topUsed-1); n > 0 {
		out = append(out, m.paneRows(n, w)...)
	}

	if h >= 4 {
		out = append(out, fit(m.moreLine(), w))
	}

	return fill(out, h)
}

// detailHeadLines names the task the view is open on: its id and title on the
// left, the repository and what the row said about it on the right.
func (m Model) detailHeadLines(w int) []string {
	t, ok := m.task(m.detail)

	left := Paint(Accent).Bold(true).Render(m.detail)
	if !ok {
		return []string{
			spread(" "+left, Paint(Dim).Render(m.opts.Words.T("detail.gone",
				"this task is no longer on the board")), w),
		}
	}

	word, role := m.stateWord(t)

	right := Paint(Dim).Render(t.Repo)
	if word != "" {
		right += Paint(Dim).Render(dot) + Paint(role).Render(word)
	}

	if t.Title == "" {
		return []string{spread(" "+left, right, w)}
	}

	rightW := lipgloss.Width(right)
	availW := max(20, w-lipgloss.Width(m.detail)-rightW-6)

	if !m.expandedDetail || lipgloss.Width(t.Title) <= availW {
		return []string{spread(" "+left+"  "+Paint(Dim).Render(t.Title), right, w)}
	}

	wrapped := splitIntoLines(t.Title, availW)

	var out []string

	out = append(out, spread(" "+left+"  "+Paint(Dim).Render(wrapped[0]), right, w))

	indent := strings.Repeat(" ", lipgloss.Width(m.detail)+3)
	for _, wl := range wrapped[1:] {
		out = append(out, " "+indent+Paint(Dim).Render(wl))
	}

	return out
}

// tabTagInfo is one tab's layout representation in the strip.
type tabTagInfo struct {
	tab      tab
	key      string
	text     string
	rendered string
	width    int
}

func (m Model) tabTags(w int) []tabTagInfo {
	names := m.tabNames()
	tags := make([]tabTagInfo, len(names))

	var fullWidth int

	for i, n := range names {
		k := paneKey(n.tab)
		tagText := "[" + k + " " + n.text + "]"

		var rend string
		if n.tab == m.tab {
			rend = Paint(Sel).Bold(true).Render(tagText)
		} else {
			rend = Paint(Dim).Render("[") + Paint(Accent).Bold(true).Render(k) +
				Paint(Dim).Render(" "+n.text+"]")
		}

		tw := lipgloss.Width(tagText)
		tags[i] = tabTagInfo{tab: n.tab, key: k, text: tagText, rendered: rend, width: tw}
		fullWidth += tw + 1
	}

	if fullWidth+10 <= w {
		return tags
	}

	var compactWidth int

	for i, n := range names {
		k := paneKey(n.tab)

		text := n.text
		if n.tab != m.tab {
			text = fit(text, 4)
		}

		tagText := "[" + k + " " + text + "]"

		var rend string
		if n.tab == m.tab {
			rend = Paint(Sel).Bold(true).Render(tagText)
		} else {
			rend = Paint(Dim).Render("[") + Paint(Accent).Bold(true).Render(k) +
				Paint(Dim).Render(" "+text+"]")
		}

		tw := lipgloss.Width(tagText)
		tags[i] = tabTagInfo{tab: n.tab, key: k, text: tagText, rendered: rend, width: tw}
		compactWidth += tw + 1
	}

	if compactWidth+10 <= w {
		return tags
	}

	for i, n := range names {
		k := paneKey(n.tab)

		tagText := "[" + k + "]"
		if n.tab == m.tab {
			tagText = "[" + k + " " + n.text + "]"
		}

		var rend string
		if n.tab == m.tab {
			rend = Paint(Sel).Bold(true).Render(tagText)
		} else {
			rend = Paint(Dim).Render("[") + Paint(Accent).Bold(true).Render(k) + Paint(Dim).Render("]")
		}

		tw := lipgloss.Width(tagText)
		tags[i] = tabTagInfo{tab: n.tab, key: k, text: tagText, rendered: rend, width: tw}
	}

	return tags
}

// tabStrip renders the eleven tabs.
func (m Model) tabStrip(w int) string {
	tags := m.tabTags(w)

	var parts []string
	for _, t := range tags {
		parts = append(parts, t.rendered)
	}

	var right string

	p := m.opts.Words
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
	tags := m.tabTags(m.frame.Body.W)
	out := make([]placedTab, len(tags))

	x := 1
	for i, t := range tags {
		out[i] = placedTab{tab: t.tab, x: x, w: t.width}
		x += t.width + 1
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
	out := []barHint{
		hintFor(m.keys.Back),
		hintFor(m.keys.NextTab),
		hint("m", m.opts.Words.T("key.tab_menu", "tab menu")),
		hint("v", m.opts.Words.T("key.toggle_markdown", "md / raw")),
		hintFor(m.keys.Ask),
		hintFor(m.keys.CLI),
	}

	wrapHint := m.opts.Words.T("detail.hint_expand", "expand")
	if m.expandedDetail {
		wrapHint = m.opts.Words.T("detail.hint_collapse", "collapse")
	}

	out = append(out, hint("e", wrapHint))
	if m.tab == tabDiff {
		out = append(out, hintFor(m.keys.Edit))
	}

	return append(out, hint(m.keys.Up.Help().Key+m.keys.Down.Help().Key,
		m.opts.Words.T("key.scroll", "scroll")))
}
