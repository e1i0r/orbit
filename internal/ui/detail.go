package ui

// The task view: one level below the board, three tabs behind one key.
//
// The board answers "what needs me". This screen answers "what happened",
// and it answers it three ways — the record, the changes, and what the
// engine actually printed. They are tabs rather than three panes side by
// side because at eighty columns three panes are three columns of twenty-six
// cells, and a diff in twenty-six cells is not a diff.
//
// The frame is four regions and the rows they take are counted in one place,
// detailTop, so that what syncPanes sizes the viewport to and what
// detailRows draws cannot drift apart: a pane one row taller than the space
// it is drawn into is a frame one row too tall, which scrolls the whole
// terminal.

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// detailRows draws the task view into the body region.
func (m Model) detailRows(h, w int) []string {
	if h <= 0 || w <= 0 {
		return nil
	}

	out := m.detailTop(h, w)

	if n := max(0, h-len(out)-1); n > 0 {
		out = append(out, m.paneRows(n, w)...)
	}

	if h >= 4 {
		out = append(out, fit(m.moreLine(), w))
	}

	return fill(out, h)
}

// detailTop is everything the pane is drawn under: the heading, the blank
// line, the tab strip, and the diff's file selector when it is up.
//
// It is its own function because where the pane starts and how many rows it
// got is a question a click on the scroll bar has to answer too, and two
// counts of the same rows are two counts that drift.
func (m Model) detailTop(h, w int) []string {
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

	return out
}

// detailHeadLines names the task the view is open on: its id and title on the
// left, the repository and what the row said about it on the right.
//
// The title is cut to one line and stripped of the inline marks it was
// written with. This is the label of the screen: a paragraph-long first line
// of task.md set here in full pushes the repository and the state off the
// right edge, and prints its own backticks besides.
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

	title := plainInline(t.Title)
	if title == "" {
		return []string{spread(" "+left, right, w)}
	}

	rightW := lipgloss.Width(right)
	availW := max(20, w-lipgloss.Width(m.detail)-rightW-6)

	if !m.expandedDetail || lipgloss.Width(title) <= availW {
		return []string{spread(" "+left+"  "+Text(Secondary).Render(fit(title, availW)), right, w)}
	}

	wrapped := splitIntoLines(title, availW)

	var out []string

	out = append(out, spread(" "+left+"  "+Text(Secondary).Render(wrapped[0]), right, w))

	indent := strings.Repeat(" ", lipgloss.Width(m.detail)+3)
	for _, wl := range wrapped[1:] {
		out = append(out, " "+indent+Text(Secondary).Render(wl))
	}

	return out
}

// tabStripReserve is the room the strip leaves at its right edge for the
// note that sits there — the attempt number, or the diff saying it had no
// base to compare against, which is the longest of them. A tier that filled
// the line would push that note off it.
const tabStripReserve = 16

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

	// Three tiers, widest first: every name in full, then names cut to four
	// cells, then keys alone with only the open tab named. The strip is one
	// line and there are eleven tabs, so what a narrow terminal drops is
	// decided here rather than by the right edge.
	for _, tier := range []func(n tabName) string{
		func(n tabName) string { return n.text },
		func(n tabName) string {
			if n.tab == m.tab {
				return n.text
			}

			return fit(n.text, 4)
		},
		func(n tabName) string {
			if n.tab == m.tab {
				return n.text
			}

			return ""
		},
	} {
		var total int

		for i, n := range names {
			k := paneKey(n.tab)
			plain, rend := tabChip(k, tier(n), n.tab == m.tab)
			tw := lipgloss.Width(plain)
			tags[i] = tabTagInfo{tab: n.tab, key: k, text: plain, rendered: rend, width: tw}
			total += tw + len(tabGap)
		}

		if total+tabStripReserve <= w {
			return tags
		}
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

	return spread(" "+strings.Join(parts, tabGap), right, w)
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
		x += t.width + len(tabGap)
	}

	return out
}

// paneRows is the pane itself, cut to the region it was given.
func (m Model) paneRows(h, w int) []string {
	vp := m.panes[m.tab]

	out := fill(strings.Split(vp.View(), "\n"), h)

	track := scrollTrack(h, vp.TotalLineCount(), vp.YOffset())
	for i, line := range out {
		if track == nil {
			out[i] = fit(line, w)
			continue
		}

		// The rail stands in the pane's last column, so the line beside it
		// is cut and filled to the column before: the viewport pads what it
		// draws out to its own width, and padding that again would put an
		// ellipsis under the rail on every row of every pane.
		beside := max(w-1, 1)
		out[i] = pad(ansi.Truncate(line, beside, ""), beside, false) + track[i]
	}

	return out
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
		hintKey("m", m.opts.Words.T("key.tab_menu", "tab menu")),
		hintKey("v", m.opts.Words.T("key.toggle_markdown", "md / raw")),
		hintFor(m.keys.Ask),
		hintFor(m.keys.CLI),
	}

	wrapHint := m.opts.Words.T("detail.hint_expand", "expand")
	if m.expandedDetail {
		wrapHint = m.opts.Words.T("detail.hint_collapse", "collapse")
	}

	out = append(out, hintKey("e", wrapHint))

	if m.tab == tabOverview {
		out = append(out, hintKey("z", m.opts.Words.T("key.fold", "fold sections")))
	}

	if m.tab == tabDiff {
		out = append(out, hintFor(m.keys.Edit))
	}

	return append(out, hint(m.keys.Up.Help().Key+m.keys.Down.Help().Key,
		m.opts.Words.T("key.scroll", "scroll")))
}
