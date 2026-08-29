package ui

import (
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// diffSummary holds counts of additions, deletions, and modified file paths.
type diffSummary struct {
	added   int
	deleted int
	files   []string
}

func parseDiffSummary(diff string) diffSummary {
	if diff == "" {
		return diffSummary{}
	}

	var sum diffSummary

	seenFiles := make(map[string]bool)

	lines := strings.Split(diff, "\n")
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, "+++ b/"):
			f := strings.TrimPrefix(l, "+++ b/")
			if f != "" && f != "/dev/null" && !seenFiles[f] {
				seenFiles[f] = true
				sum.files = append(sum.files, f)
			}
		case strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++"):
			sum.added++
		case strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---"):
			sum.deleted++
		}
	}

	return sum
}

// overviewLines renders Pane 1: what became of this task, in the order a
// reader asks for it — what it is, whether it wants something from them, what
// it spent, what the model said, what changed on disk, and what can be done
// with it now.
func (m Model) overviewLines() []string {
	p := m.opts.Words

	if m.logErr != nil {
		return []string{paneGutter + Paint(Bad).Render(m.errSaid(m.logErr))}
	}

	t, ok := m.task(m.detail)
	if !ok {
		return []string{paneGutter + Paint(Dim).Render(p.T("detail.gone", "this task is no longer on the board"))}
	}

	w := max(40, m.frame.Body.W)

	out := []string{""}
	out = append(out, m.overviewHead(t, w)...)
	out = append(out, m.overviewVitals(t, w)...)
	out = append(out, m.overviewPhases(t, w)...)
	out = append(out, m.overviewChanges(w)...)

	return append(out, m.overviewActions(w)...)
}

// overviewHead opens the pane with what this task is: the id, the repository
// and the state on one line, and under it the task in the words it was
// written in.
//
// The title is not set again here. The header two lines above already
// carries it, and a first line of task.md that runs to a paragraph was
// drawn twice on one screen — once bold, once dim — which is a wall to read
// past rather than a title to read.
func (m Model) overviewHead(t view.Task, w int) []string {
	p := m.opts.Words
	word, role := m.stateWord(t)

	out := []string{paneGutter + meta(
		Text(Secondary).Render(t.ID),
		Text(Secondary).Render(t.Repo),
		badge(m.bandGlyph(t)+" "+word, role),
	), ""}

	out = append(out, m.overviewBrief(w)...)

	if role != Warn && role != Bad {
		return out
	}

	// The one banner in the window. A task that wants the reader has to say
	// so louder than the four figures under it, or it waits until they scroll.
	return append(out,
		paneGutter+Paint(role).Bold(true).Render("▍ "+p.T("overview.waiting_box", "NEEDS YOU")),
		paneGutter+Text(Tertiary).Render(p.T("overview.resume_hint", "press 't' to open interactive session, 'r' to restart")),
		"",
	)
}

// overviewBriefRows is how much of the brief a closed pane shows: enough to
// know what was asked for, and short enough that the figures under it are
// still on the screen. The rest is one keystroke away.
const overviewBriefRows = 8

// overviewBrief is the task as its author wrote it. The first line of
// task.md is the title; everything under it is the brief, and nothing in the
// window drew it — the pane opened on figures about a task whose own words
// were on disk and nowhere else.
//
// A task written without one draws nothing here: the header names it, and
// the whole point of taking the title off this pane was to stop the same
// paragraph being set twice on one screen.
func (m Model) overviewBrief(w int) []string {
	for _, e := range m.entries {
		if e.What() != view.EntryWritten {
			continue
		}

		_, body, _ := strings.Cut(e.Text, "\n")

		rows := renderMarkdown(strings.TrimSpace(body), w, m.rawText)
		if len(rows) == 0 {
			return nil
		}

		if !m.expandedDetail && len(rows) > overviewBriefRows {
			return append(rows[:overviewBriefRows], markdownIndent+Text(Tertiary).Render(
				m.opts.Words.T("overview.more", "… [e] for all of it")), "")
		}

		return append(rows, "")
	}

	return nil
}

// bandGlyph is the mark that stands before the state word: a shape for
// readers who cannot tell the four state colours apart, and a spinner while
// something is actually happening.
func (m Model) bandGlyph(t view.Task) string {
	switch t.Band {
	case view.Done:
		return "✓"
	case view.Running:
		return strings.TrimSpace(m.runGlyph(working(t)))
	case view.NeedsYou:
		return "▲"
	case view.ToDo:
		return "○"
	}

	return "○"
}
