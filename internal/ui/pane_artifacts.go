package ui

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
	"maps"
	"path/filepath"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
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
		Paint(Accent).Render(pad(name, fileNameCells, false)) + "  " +
		Paint(Dim).Render(pad(size, fileSizeCells, false)) + "  " +
		Paint(Dim).Render(fit(said, max(w-fileRowLead, 8)))
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
func (m Model) fileSaid(name string) string {
	p := m.opts.Words

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
func (m Model) bodyCells() int { return max(m.frame.Body.W, 1) }

// artifactsLines is the artifacts tab's content, ready for the pane.
func (m Model) artifactsLines() []string {
	lines, _ := m.artifactsRows()

	return lines
}

// artifactsRows is that content and, beside it, which file each row that
// folds is the head of.
func (m Model) artifactsRows() ([]string, map[int]int) {
	p := m.opts.Words

	if _, ok := m.task(m.detail); !ok {
		return []string{"  " + Paint(Dim).Render(p.T("detail.gone", "this task is no longer on the board"))}, nil
	}

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("artifacts.title", "Files & Artifacts")),
		"  " + Paint(Dim).Render(p.T("artifacts.subtitle", "every file the run left, and what each one is")),
		"",
	}

	heads := map[int]int{}
	out = m.recordFiles(out, heads)

	out = append(out, "")
	out = append(out, m.worktreeFiles()...)
	out = append(out, "")

	return out, heads
}

// recordFiles is what Orbit itself wrote about the run, one row per file and
// what that file holds under whichever of them the reader has opened.
func (m Model) recordFiles(out []string, heads map[int]int) []string {
	p := m.opts.Words

	out = append(out, m.artifactsHead(p.T("artifacts.group_record", "what orbit wrote down"),
		p.P("artifacts.n_files", len(m.files), "{n} file", "{n} files")))

	switch {
	case m.filesErr != nil:
		return append(out, "    "+Paint(Bad).Render(m.errSaid(m.filesErr)))
	case !m.filesKnown:
		return append(out, "    "+Paint(Dim).Render(p.T("artifacts.reading", "reading the task's directory")))
	case len(m.files) == 0:
		return append(out, "    "+Paint(Dim).Render(p.T("artifacts.none_yet", "nothing written yet — this task has not run")))
	}

	for i, f := range m.files {
		open := m.rowOpen(tabArtifacts, i)

		heads[len(out)] = i
		out = append(out, "  "+Text(Tertiary).Render(foldMark(open))+
			fileRow(f.Name, formatBytes(f.Size), m.fileSaid(f.Name), m.bodyCells()-4))

		if open {
			out = append(out, m.fileBody(f.Name)...)
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
func (m Model) fileBody(name string) []string {
	p := m.opts.Words
	w := max(m.bodyCells()-6, 20)

	got, asked := m.read[name]

	switch {
	case !asked:
		return []string{"      " + Paint(Dim).Render(p.T("artifacts.opening", "opening the file"))}
	case got.err != nil:
		return []string{"      " + Paint(Bad).Render(m.errSaid(got.err))}
	case strings.TrimSpace(got.text.Text) == "":
		return []string{"      " + Paint(Dim).Render(p.T("artifacts.empty_file", "this file is empty"))}
	}

	lines := strings.Split(strings.TrimRight(got.text.Text, "\n"), "\n")

	out := make([]string, 0, len(lines)+1)
	for _, l := range lines {
		out = append(out, "      "+codeWell(l, fileFamily(name), w))
	}

	if !got.text.Whole {
		out = append(out, "      "+Text(Tertiary).Render(
			p.T("artifacts.cut", "— the rest of this file was not read —")))
	}

	return out
}

// fileFamily is the syntax a file of the task's directory is read with,
// taken from its name: the two Orbit writes without an extension are a word
// and a pair of fields, and neither is anybody's language.
func fileFamily(name string) string {
	ext := strings.TrimPrefix(filepath.Ext(name), ".")

	return codeFamily(ext)
}

// worktreeFiles is what the run wrote about the repository, which is the
// diff and is counted from it: the worktree is a checkout Orbit does not
// keep, and the file that is not in the diff is the file the run left alone.
//
// The rows do not fold. What is in them is the diff, it is a tab of its own,
// and a second rendering of it here would be a second place to keep right.
func (m Model) worktreeFiles() []string {
	p := m.opts.Words

	var changed []string

	if m.diffKnown && m.diff != "" {
		changed = parseDiffSummary(m.diff).files
	}

	head := m.artifactsHead(p.T("artifacts.group_worktree", "what the run changed"),
		p.P("artifacts.n_files", len(changed), "{n} file", "{n} files"))

	switch {
	case m.diffErr != nil:
		return []string{head, "    " + Paint(Bad).Render(m.errSaid(m.diffErr))}
	case !m.diffKnown:
		return []string{head, "    " + Paint(Dim).Render(p.T("artifacts.reading_worktree", "reading the worktree"))}
	case len(changed) == 0:
		return []string{head, "    " + Paint(Dim).Render(p.T("diff.unchanged", "no changes in this task's worktree"))}
	}

	out := []string{head}
	for _, f := range changed {
		out = append(out, fileRow(f, "", p.T("artifacts.said_changed", "changed in the worktree — see the diff tab"), m.bodyCells()))
	}

	return out
}

// artifactsHead is one section's heading: what the section is and how much
// is under it. It does not fold — the rows under it do, and a lid over the
// lids would put the thing a reader came for two gestures away.
func (m Model) artifactsHead(label, count string) string {
	return "  " + Paint(Accent).Render(label) + "  " + Paint(Dim).Render(count)
}

// fileRead is what one file turned out to hold, or why it could not be read.
// The two are one type because a row shows one or the other and never both.
type fileRead struct {
	text view.FileText
	err  error
}

// readFile writes down what a file turned out to hold.
//
// The map is cloned rather than written in place, for the reason every other
// map on the model is: a Model is copied by every method that returns one
// and a map is not.
func (m Model) readFile(msg fileTextMsg) Model {
	held := maps.Clone(m.read)
	if held == nil {
		held = map[string]fileRead{}
	}

	held[msg.Name] = fileRead{text: msg.Text, err: msg.Err}
	m.read = held

	return m.syncPanes()
}
