package ui

// Where the window and the twelve panes meet.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/panes"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// panesEnv is the run being read, as the panes were written to read it.
// What is folded and what is on screen is the window's and is handed over;
// what the record says is the record's.
func (m Model) panesEnv() panes.Env {
	t, ok := m.task(m.detail)

	f, flowErr := m.taskFlow(t)
	word, role := m.stateWord(t)

	return panes.Env{
		Words:       m.opts.Words,
		Frame:       m.frame,
		Now:         m.now,
		Task:        t,
		Gone:        !ok,
		Entries:     m.entries,
		Flow:        f,
		FlowFailed:  flowErr,
		Failed:      m.errSaid(m.logErr),
		Word:        word,
		Role:        role,
		Live:        m.runGlyph(keymap.Working(t)),
		Keys:        m.keys,
		Expanded:    m.expandedDetail,
		Diff:        m.diff,
		DiffKnown:   m.diffKnown,
		DiffFailed:  m.errSaid(m.diffErr),
		DiffMissing: gitLostTheWorktree(m.diffErr),
		Width:       m.width,
		Collapsed:   m.collapsedFiles,
		Rationale:   !m.hideDiffRationale,
		Files:       m.files,
		FilesKnown:  m.filesKnown,
		FilesFailed: m.errSaid(m.filesErr),
		Read:        m.fileHeld,
		Reach:       m.reading(),
		Shape:       m.shaping(),
		Said:        m.errSaid,
		Spinner:     m.spinner(theme.Live),
		Dials:       m.taskDials(t),
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
	// The world syncPanes built for this pass, when there is one. Each pane
	// asking for its own resolved the flow twice and asked the engines port,
	// so twelve panes read the flow twenty-four times where one reading
	// does — on every log line, every diff and every fold. Outside that pass
	// m.world is nil and this builds its own, because a reading of an
	// earlier instant would draw a task that has since moved on.
	e := m.panesEnv()
	if m.world != nil {
		e = *m.world
	}

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

// historyRows is every word said about this task, in any program.
func (m Model) historyRows() ([]string, map[int]int) {
	return panes.History(m.paneEnv(tabHistory))
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

// taskFlow is the pipeline a task was started under, resolved: reading the
// flow store is the window's door and not a pane's.
func (m Model) taskFlow(t view.Task) (flow.Flow, string) {
	name := cells.OrDef(t.Flow, flow.Default)

	f, err := flow.Resolve(m.opts.Flows, name)
	if err != nil {
		return flow.Flow{}, fmt.Sprintf("flow %q: %v", name, err)
	}

	return f, ""
}

// flowLines is the pipeline this run was started under, drawn as a tree.
func (m Model) flowLines() []string {
	lines, _ := m.flowRows()

	return lines
}

// flowRows is that tree and which phase each node that folds stands for.
func (m Model) flowRows() ([]string, map[int]int) {
	return panes.Pipeline(m.paneEnv(tabFlow))
}

// overviewLines is what became of this task, in the order a reader asks for
// it.
func (m Model) overviewLines() []string { return panes.Overview(m.paneEnv(tabOverview)) }

// taskDials is what a task would run on: what it carries where it has run,
// and what the knobs say where it has not — read through the engines port,
// which is a door of the window's and not a pane's.
func (m Model) taskDials(t view.Task) panes.Dials {
	// Not the words claude and sonnet, which were the answer here on builds
	// that have neither: a window whose engines port answers nothing has no
	// engine and no model to name, and a dash says so.
	eng := cells.OrDef(t.Engine, m.dialEngine(m.knobs.Engine))

	models, _ := m.modelsFor(eng)
	mod := cells.OrDef(t.Model, cells.OrDef(m.knobs.Model, cells.First(models)))

	return panes.Dials{
		Engine:   cells.OrDef(eng, unsetDial),
		Model:    cells.OrDef(mod, unsetDial),
		Effort:   m.knobs.Effort,
		Thinking: m.knobs.Thinking,
	}
}

// fileHeld is what an opened file turned out to hold, in the words the pane
// draws: the window says an error, and a pane is handed the sentence.
func (m Model) fileHeld(name string) (panes.File, bool) {
	got, asked := m.read[name]
	if !asked {
		return panes.File{}, false
	}

	return panes.File{Text: got.text.Text, Whole: got.text.Whole, Failed: m.errSaid(got.err)}, true
}

// artifactsLines is every file the run left, and what each one is.
func (m Model) artifactsLines() []string {
	lines, _ := m.artifactsRows()

	return lines
}

// artifactsRows is that content and which file each row that folds is the
// head of.
func (m Model) artifactsRows() ([]string, map[int]int) {
	return panes.Artifacts(m.paneEnv(tabArtifacts))
}

// gitLostTheWorktree matches git's own words, before they are translated: a
// checkout that is not there is recognised by what git said, and what git
// said is the same sentence whatever language this window is drawn in.
func gitLostTheWorktree(err error) bool {
	if err == nil {
		return false
	}

	said := err.Error()

	return strings.Contains(said, "cannot change to") || strings.Contains(said, "no such file or directory")
}

// diffLines is what the task changed.
func (m Model) diffLines() []string {
	lines, _ := m.diffRows()

	return lines
}

// diffRows is that content and where each file's card landed.
func (m Model) diffRows() ([]string, map[int]int) {
	return panes.Diff(m.paneEnv(tabDiff))
}

// reading is what the window last read about what this change touches, in
// the words the pane draws it in.
func (m Model) reading() panes.Reach {
	t, held := m.task(m.detail)

	out := panes.Reach{
		History:      m.weigh.reach,
		Read:         m.weigh.reachKnown,
		Asking:       m.weigh.reachAsking,
		Failed:       m.errSaid(m.weigh.reachErr),
		NoHistory:    m.opts.Reader == nil,
		Checks:       m.weigh.checks,
		Compared:     m.weigh.checksKnown,
		ChecksFailed: m.errSaid(m.weigh.checksErr),
		Running:      m.weigh.running,
		For:          m.comparedFor(),
	}

	if held {
		out.Offered = m.checksOf(t)
	}

	return out
}

// shaping is the repository as a tree, in the words the map pane draws it in.
func (m Model) shaping() panes.Shape {
	return panes.Shape{
		Tree:    m.shape.tree,
		Read:    m.shape.known,
		Asking:  m.shape.asking,
		Failed:  m.errSaid(m.shape.err),
		Missing: m.shape.missing,
	}
}

// impactRows is what this change reaches beyond the files it touched.
func (m Model) impactRows() []string { return panes.Impact(m.paneEnv(tabImpact)) }

// mapRows is the repository as a tree, lit where this task changed it.
func (m Model) mapRows() []string { return panes.Map(m.paneEnv(tabMap)) }

// impactMark is the count beside the impact tab's name.
//
// Built from the three fields Mark actually reads and not from the whole
// pane world. tabNames calls this on every draw of the detail screen and on
// every mouse motion over the tab strip, and panesEnv resolves the flow
// twice and asks the engines port on its way to one number — which a task
// with a running spinner paid for twice per animation frame.
func (m Model) impactMark() string {
	return panes.Mark(panes.Env{Reach: panes.Reach{
		History: m.weigh.reach,
		Read:    m.weigh.reachKnown,
		Failed:  m.errSaid(m.weigh.reachErr),
	}})
}
