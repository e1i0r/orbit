package panes

// The diff tab: what the task changed, and the one key that leaves the
// window for an editor.
//
// It is git's own output, coloured on the two characters that carry the
// meaning and nothing else. There is no syntax highlighting and there will
// not be: it would mean a lexer per language in a terminal cockpit, and the
// question this pane answers — what did the agent change — is answered by
// the plus and the minus.
//
// Long lines scroll rather than wrap. A diff of a generated file arrives as
// one line of several thousand cells, and wrapping it would push the rest of
// the hunk off the bottom of the screen; the pane cuts, and ← and → move
// along it.
//
// Running git and waiting for it is gitdiff.go's job, not this file's: what
// is here reads the answer once there is one — which file a line belongs
// to, what colour a line is, and which file and line the editor opens.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/patch"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// Diff is what the task changed, and where each file's card landed.
//
// It answers in one of four states, because there are four different true
// things it can say. Before the first answer lands there is nothing read
// yet, and that is not the same fact as an answer that came back empty —
// collapsing the two, which this pane once did, is how a git that hangs
// ends up asserting "no changes" on a question it was never actually
// answered. A worktree that is gone and a git that broke are the other two.
func Diff(e Env) ([]string, map[int]int) {
	p := e.Words

	if !e.Gone && view.BandOf(e.Task) == view.ToDo {
		return []string{" " + theme.Paint(theme.Dim).Render(p.T("diff.empty_todo",
			"no changes yet — task is in the todo queue (press [n] to start)"))}, nil
	}

	switch {
	case !e.DiffKnown:
		return []string{" " + theme.Paint(theme.Dim).Render(
			p.T("diff.pending", "reading this task's worktree…"))}, nil
	case e.DiffMissing:
		return []string{" " + theme.Paint(theme.Dim).Render(
			p.T("diff.empty_no_worktree", "no working tree modifications recorded"))}, nil
	case e.DiffFailed != "":
		// Folded to the pane rather than drawn as one line. What follows
		// the sentence is evidence — the command that timed out and the
		// worktree it ran in — and errSaid keeps it on purpose, so a git
		// that hung wrote a path across a pane three lines tall.
		return failedLines(e.DiffFailed, e.Width), nil
	case strings.TrimSpace(e.Diff) == "":
		return []string{" " + theme.Paint(theme.Dim).Render(
			p.T("diff.unchanged", "no changes in this task's worktree"))}, nil
	}

	files := patch.Files(strings.Split(strings.TrimSuffix(e.Diff, "\n"), "\n"))
	rationales := patch.Rationales(e.Entries, files, p)
	lines, drawn := formatStructuredDiff(e.Diff, e.Width, p, rationales, e.Rationale, e.Collapsed, e.Expanded)

	// Where each card landed, taken from the pass that drew it rather than
	// counted again afterwards: a second count is a second opinion about
	// where a row is, and the day they disagree the pointer collapses the
	// file above the one it is on.
	heads := make(map[int]int, len(drawn))
	for i, f := range drawn {
		heads[f.StartLine] = i
	}

	return lines, heads
}

// failedLines is a refusal folded into the pane it is drawn in.
func failedLines(said string, width int) []string {
	var out []string

	for _, line := range cells.Lines(said, max(20, width-2)) {
		out = append(out, " "+theme.Paint(theme.Bad).Render(line))
	}

	return out
}
