package panes

// The artifacts tab: what the run left on disk.
//
// Every row here is read rather than composed. The pane this replaces listed
// files the store has never written — gates.json, task.env, cost.tsv, state
// — beside sizes typed into the source, and a listing that cannot be checked
// against the disk is worse than no listing: it is the one screen a reader
// would go to precisely to check.
//
// The two halves are two different questions. The task's own directory is
// what Orbit wrote about the run; the worktree is what the run wrote about
// the repository, and that half is the diff, counted rather than measured.

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/markdown"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// fileNameCells is the tab's name column and fileSizeCells the measure
// beside it.
const (
	fileNameCells = 28
	fileSizeCells = 8
)

// fileRowLead is what a row spends before its sentence: four of indent, the
// two columns, and the two gaps between the three.
const fileRowLead = 4 + fileNameCells + 2 + fileSizeCells + 2

// fileRow is one file: what it is called, how big it is, and what it holds,
// in w cells.
//
// The columns are padded on the plain text and painted afterwards. A width
// verb counts the bytes of an escape sequence as characters, so a rendered
// string padded to a column is not padded at all — which is why this listing
// has never lined up.
//
// The sentence is cut to what is left rather than allowed to run on: the
// pane does not wrap, so a row wider than it is a row the terminal decides
// the end of, without the ellipsis that says a decision was made.
func fileRow(name, size, said string, w int) string {
	return "    " +
		theme.Paint(theme.Accent).Render(cells.Pad(name, fileNameCells, false)) + "  " +
		theme.Paint(theme.Dim).Render(cells.Pad(size, fileSizeCells, false)) + "  " +
		theme.Paint(theme.Dim).Render(cells.Fit(said, max(w-fileRowLead, 8)))
}

// formatBytes is a size in the largest unit that keeps it a whole number.
func formatBytes(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%d k", bytes/1024)
	}

	return fmt.Sprintf("%d M", bytes/(1024*1024))
}

// fileSaid is what a file of the task's directory is for.
//
// A name this build does not know is described as nothing at all rather than
// guessed at: the name and the size are read from the disk and are true, and
// a sentence invented beside them would be the one part of the row that is
// not.
func (e Env) fileSaid(name string) string {
	p := e.Words

	switch name {
	case "task.md":
		return p.T("artifacts.said_task", "the task as it was written, in full")
	case "events.jsonl":
		return p.T("artifacts.said_events", "the append-only record: one line per event")
	case "control":
		return p.T("artifacts.said_control", "the word the run was last told: pause, resume, cancel")
	case "run":
		return p.T("artifacts.said_run", "the marker that says a run holds this task")
	}

	return ""
}

// bodyCells is the pane's width, never nought: a window that has not been
// told its size yet still draws a frame.
func (e Env) bodyCells() int { return max(e.Frame.Body.W, 1) }

// Artifacts is every file the run left, and what each one is: what Orbit
// wrote about the run, and what the run wrote about the repository. Beside
// it is which file each row that folds is the head of.
func Artifacts(e Env) ([]string, map[int]int) {
	p := e.Words

	if e.Gone {
		return []string{"  " + theme.Paint(theme.Dim).Render(
			p.T("detail.gone", "this task is no longer on the board"))}, nil
	}

	out := []string{
		"",
		"  " + theme.Paint(theme.Accent).Bold(true).Render(p.T("artifacts.title", "Files & Artifacts")),
		"  " + theme.Paint(theme.Dim).Render(p.T("artifacts.subtitle", "every file the run left, and what each one is")),
		"",
	}

	heads := map[int]int{}
	out = e.recordFiles(out, heads)

	out = append(out, "")
	out = append(out, e.worktreeFiles()...)
	out = append(out, "")

	return out, heads
}

// recordFiles is what Orbit itself wrote about the run, one row per file and
// what that file holds under whichever of them the reader has opened.
func (e Env) recordFiles(out []string, heads map[int]int) []string {
	p := e.Words

	out = append(out, artifactsHead(p.T("artifacts.group_record", "what orbit wrote down"),
		p.P("artifacts.n_files", len(e.Files), "{n} file", "{n} files")))

	switch {
	case e.FilesFailed != "":
		return append(out, "    "+theme.Paint(theme.Bad).Render(e.FilesFailed))
	case !e.FilesKnown:
		return append(out, "    "+theme.Paint(theme.Dim).Render(
			p.T("artifacts.reading", "reading the task's directory")))
	case len(e.Files) == 0:
		return append(out, "    "+theme.Paint(theme.Dim).Render(
			p.T("artifacts.none_yet", "nothing written yet — this task has not run")))
	}

	for i, f := range e.Files {
		open := e.row(i)

		heads[len(out)] = i
		out = append(out, "  "+theme.Text(theme.Tertiary).Render(cells.Fold(open))+
			fileRow(f.Name, formatBytes(f.Size), e.fileSaid(f.Name), e.bodyCells()-4))

		if open {
			out = append(out, e.fileBody(f.Name)...)
		}
	}

	return out
}

// fileBody is what an opened file holds, as it is on disk.
//
// It is not wrapped. A record is one event per line and a marker is one
// field per line, and a line folded onto the next would read as two of them
// — so a line too long for the pane is cut by the well it is drawn in, and
// the reader who needs the rest of it opens the file.
func (e Env) fileBody(name string) []string {
	p := e.Words
	w := max(e.bodyCells()-6, 20)

	got, asked := e.read(name)

	switch {
	case !asked:
		return []string{"      " + theme.Paint(theme.Dim).Render(p.T("artifacts.opening", "opening the file"))}
	case got.Failed != "":
		return []string{"      " + theme.Paint(theme.Bad).Render(got.Failed)}
	case strings.TrimSpace(got.Text) == "":
		return []string{"      " + theme.Paint(theme.Dim).Render(p.T("artifacts.empty_file", "this file is empty"))}
	}

	lines := strings.Split(strings.TrimRight(got.Text, "\n"), "\n")

	out := make([]string, 0, len(lines)+1)
	for _, l := range lines {
		out = append(out, "      "+markdown.Well(l, fileFamily(name), w))
	}

	if !got.Whole {
		out = append(out, "      "+theme.Text(theme.Tertiary).Render(
			p.T("artifacts.cut", "— the rest of this file was not read —")))
	}

	return out
}

// fileFamily is the syntax a file of the task's directory is read with,
// taken from its name: the two Orbit writes without an extension are a word
// and a pair of fields, and neither is anybody's language.
func fileFamily(name string) string {
	ext := strings.TrimPrefix(filepath.Ext(name), ".")

	return theme.CodeFamily(ext)
}

// worktreeFiles is what the run wrote about the repository, which is the
// diff and is counted from it: the worktree is a checkout Orbit does not
// keep, and the file that is not in the diff is the file the run left alone.
//
// The rows do not fold. What is in them is the diff, it is a tab of its own,
// and a second rendering of it here would be a second place to keep right.
func (e Env) worktreeFiles() []string {
	p := e.Words

	var changed []string

	if e.DiffKnown && e.Diff != "" {
		changed = Changed(e.Diff)
	}

	head := artifactsHead(p.T("artifacts.group_worktree", "what the run changed"),
		p.P("artifacts.n_files", len(changed), "{n} file", "{n} files"))

	switch {
	case e.DiffFailed != "":
		return []string{head, "    " + theme.Paint(theme.Bad).Render(e.DiffFailed)}
	case !e.DiffKnown:
		return []string{head, "    " + theme.Paint(theme.Dim).Render(
			p.T("artifacts.reading_worktree", "reading the worktree"))}
	case len(changed) == 0:
		return []string{head, "    " + theme.Paint(theme.Dim).Render(p.T("diff.unchanged", "no changes in this task's worktree"))}
	}

	out := []string{head}
	for _, f := range changed {
		out = append(out, fileRow(f, "", p.T("artifacts.said_changed", "changed in the worktree — see the diff tab"), e.bodyCells()))
	}

	return out
}

// artifactsHead is one section's heading: what the section is and how much
// is under it. It does not fold — the rows under it do, and a lid over the
// lids would put the thing a reader came for two gestures away.
func artifactsHead(label, count string) string {
	return "  " + theme.Paint(theme.Accent).Render(label) + "  " + theme.Paint(theme.Dim).Render(count)
}
