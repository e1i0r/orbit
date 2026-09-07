package panes

import (
	"strings"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/markdown"
	"github.com/e1i0r/orbit/internal/ui/prose"
	"github.com/e1i0r/orbit/internal/ui/theme"
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

// Overview is what became of this task, in the order a reader asks for it —
// what it is, whether it wants something from them, what it spent, what the
// model said, what changed on disk, and what can be done with it now.
func Overview(e Env) []string {
	p := e.Words

	if e.Failed != "" {
		return []string{prose.Gutter + theme.Paint(theme.Bad).Render(e.Failed)}
	}

	if e.Gone {
		return []string{prose.Gutter + theme.Paint(theme.Dim).Render(
			p.T("detail.gone", "this task is no longer on the board"))}
	}

	t, w := e.Task, max(40, e.Frame.Body.W)

	out := []string{""}
	out = append(out, e.head(t, w)...)
	out = append(out, e.storyLines(w)...)
	out = append(out, e.vitals(t, w)...)
	out = append(out, e.phases(t, w)...)
	out = append(out, e.changes(w)...)

	return append(out, e.actions(w)...)
}

// FoldRows is which row of the pane each section head landed on.
//
// A head is the first row of its block, so where it landed is the count of
// what the pane drew before it. The blocks are asked their own heights here
// rather than searched for in the drawn rows, which keeps a translated label
// out of a hit test — and pins this arithmetic to Overview, which is what
// the test beside it is for.
//
// The story is counted with the rest. It was not, and a task that wrote one
// put every section head that many rows below where a click looked for it.
func FoldRows(e Env) map[int]string {
	if e.Gone || e.Failed != "" {
		return nil
	}

	t, w := e.Task, max(40, e.Frame.Body.W)

	// The blank line Overview opens the pane with.
	at := 1 + len(e.head(t, w)) + len(e.storyLines(w)) + len(e.vitals(t, w))

	rows := map[int]string{at: FoldPhases}

	at += len(e.phases(t, w))
	rows[at] = FoldChanges

	at += len(e.changes(w))
	rows[at] = FoldDeliver

	return rows
}

// head opens the pane with what this task is: the id, the repository
// and the state on one line, and under it the task in the words it was
// written in.
//
// The title is not set again here. The header two lines above already
// carries it, and a first line of task.md that runs to a paragraph was
// drawn twice on one screen — once bold, once dim — which is a wall to read
// past rather than a title to read.
func (e Env) head(t view.Task, w int) []string {
	p := e.Words

	out := []string{prose.Gutter + prose.Meta(
		theme.Text(theme.Secondary).Render(t.ID),
		theme.Text(theme.Secondary).Render(t.Repo),
		prose.Badge(e.bandGlyph(t)+" "+e.Word, e.Role),
	), ""}

	out = append(out, e.brief(w)...)

	if e.Role != theme.Warn && e.Role != theme.Bad {
		return out
	}

	// The one banner in the window. A task that wants the reader has to say
	// so louder than the four figures under it, or it waits until they scroll.
	//
	// The keys are asked for rather than written into the sentence. This
	// line read "press 't'", and t on this screen is the thinking dial: the
	// reader who did as it said turned thinking off and got no session. A
	// letter written into a sentence is a letter nothing keeps true.
	return append(out,
		prose.Gutter+theme.Paint(e.Role).Bold(true).Render("▍ "+p.T("overview.waiting_box", "NEEDS YOU")),
		prose.Gutter+theme.Text(theme.Tertiary).Render(e.waitingHint(t)),
		"",
	)
}

// waitingHint is the line under NEEDS YOU: the two things that are always
// worth doing to a task that is waiting, and the one that sets it going
// again.
//
// Which key that last one is follows the task. Resume is for a run stopped
// at a phase boundary, and a task that failed, timed out or was abandoned
// has no process left to let go of: this line named resume anyway, so the
// reader who did as it said was told that resuming needs a paused task —
// having been sent there by the window itself. What such a task takes is a
// fresh run, which is the start dialog.
func (e Env) waitingHint(t view.Task) string {
	p := e.Words

	if keymap.WhyNotResume(t).Name != "" {
		return p.T("overview.start_hint",
			"press '{cli}' to open an interactive session, '{ask}' to leave feedback, '{start}' to start a run",
			about("cli", e.Keys.CLI.Help().Key),
			about("ask", e.Keys.Ask.Help().Key),
			about("start", e.Keys.Start.Help().Key))
	}

	return p.T("overview.resume_hint",
		"press '{cli}' to open an interactive session, '{ask}' to leave feedback, '{resume}' to resume",
		about("cli", e.Keys.CLI.Help().Key),
		about("ask", e.Keys.Ask.Help().Key),
		about("resume", e.Keys.Resume.Help().Key))
}

// overviewBriefRows is how much of the brief a closed pane shows: enough to
// know what was asked for, and short enough that the figures under it are
// still on the screen. The rest is one keystroke away.
const overviewBriefRows = 8

// brief is the task as its author wrote it. The first line of
// task.md is the title; everything under it is the brief, and nothing in the
// window drew it — the pane opened on figures about a task whose own words
// were on disk and nowhere else.
//
// A task written without one draws nothing here: the header names it, and
// the whole point of taking the title off this pane was to stop the same
// paragraph being set twice on one screen.
func (e Env) brief(w int) []string {
	for _, entry := range e.Entries {
		if entry.What() != view.EntryWritten {
			continue
		}

		_, body, _ := strings.Cut(entry.Text, "\n")

		rows := markdown.Render(strings.TrimSpace(body), w, e.Raw)
		if len(rows) == 0 {
			return nil
		}

		if !e.Expanded && len(rows) > overviewBriefRows {
			return append(rows[:overviewBriefRows], markdown.Indent+theme.Text(theme.Tertiary).Render(
				e.Words.T("overview.more", "… [e] for all of it")), "")
		}

		return append(rows, "")
	}

	return nil
}

// bandGlyph is the mark that stands before the state word: a shape for
// readers who cannot tell the four state colours apart, and a spinner while
// something is actually happening.
func (e Env) bandGlyph(t view.Task) string {
	switch t.Band {
	case view.Done:
		return "✓"
	case view.Running:
		return strings.TrimSpace(e.Live)
	case view.NeedsYou:
		return "▲"
	case view.ToDo:
		return "○"
	}

	return "○"
}
