package ui

// Where the window and the twelve panes meet.

import (
	"github.com/e1i0r/orbit/internal/ui/panes"
)

// panesEnv is the run being read, as the panes were written to read it.
// What is folded and what is on screen is the window's and is handed over;
// what the record says is the record's.
func (m Model) panesEnv() panes.Env {
	t, ok := m.task(m.detail)

	return panes.Env{
		Words:       m.opts.Words,
		Frame:       m.frame,
		Now:         m.now,
		Task:        t,
		Gone:        !ok,
		Entries:     m.entries,
		Failed:      m.errSaid(m.logErr),
		Raw:         m.rawText,
		Priced:      m.spends(t.Engine),
		Folded:      m.folded,
		AttemptOpen: m.attemptOpen,
	}
}

// paneEnv is that world with one pane's own fold answers in it: a pane is
// drawn whether or not it is the one on screen, so the rows being asked
// about are never assumed to be the rows in front of the reader.
func (m Model) paneEnv(t tab) panes.Env {
	e := m.panesEnv()
	e.RowOpen = func(i int) bool { return m.rowOpen(t, i) }

	return e
}

// costLines is what has been spent, stage by stage.
func (m Model) costLines() []string { return panes.Cost(m.paneEnv(tabCost)) }

// thinkingLines is the reasoning the engine showed its work in.
func (m Model) thinkingLines() []string {
	lines, _ := m.thinkingRows()

	return lines
}

// thinkingRows is that content and which entry each block that folds was
// written by.
func (m Model) thinkingRows() ([]string, map[int]int) {
	return panes.Thinking(m.paneEnv(tabThinking))
}

// refusedLines is what the sandbox would not let this run do.
func (m Model) refusedLines() []string {
	lines, _ := m.refusedRows()

	return lines
}

// refusedRows is that content and which denial each row that folds stands
// for.
func (m Model) refusedRows() ([]string, map[int]int) {
	return panes.Refused(m.paneEnv(tabRefused))
}

// gatesLines is what has to pass before a phase stands.
func (m Model) gatesLines() []string {
	lines, _ := m.gatesRows()

	return lines
}

// gatesRows is that content and which check each row that folds stands for.
func (m Model) gatesRows() ([]string, map[int]int) {
	return panes.Gates(m.paneEnv(tabGates))
}

// logLines is the record of this task, oldest first.
func (m Model) logLines() []string { return m.logRows().Rows }

// logRows is that content and which entry each row that folds is the head
// of, and which attempt each seam belongs to.
func (m Model) logRows() panes.Drawn {
	return panes.Timeline(m.paneEnv(tabTimeline))
}

// reportLines is what the engine wrote about the change.
func (m Model) reportLines() []string {
	lines, _ := m.reportRows()

	return lines
}

// reportRows is that content and which attempt each seam belongs to.
func (m Model) reportRows() ([]string, map[int]int) {
	return panes.Report(m.paneEnv(tabReport))
}

// notesLines is everything spoken with the model about this task.
func (m Model) notesLines() []string {
	lines, _ := m.notesRows()

	return lines
}

// notesRows is that content and which item each row that folds stands for.
func (m Model) notesRows() ([]string, map[int]int) {
	return panes.Notes(m.paneEnv(tabNotes))
}
