package ui

// What each key does, in one sentence.
//
// why.go says why a verb is refused here; this says what the verb is for at
// all, which is the question a reader who has never pressed it has. They are
// longer than the descriptions in the key map on purpose: that one is a
// label on a bar with a column budget, and this one has the width of the
// band to itself.
//
// Every key is a literal at its call site for the reason why.go's are:
// internal/words checks the catalogue against these sources statically, and
// it can only do that for a literal.

import (
	"fmt"

	"charm.land/bubbles/v2/key"
)

// meaning is the sentence for one keystroke, and it is never empty: a key
// nothing is bound to still gets an answer, because that it does nothing is
// what the reader asked about.
//
// The order is the key map's, so that the two are read side by side. Where
// two bindings share a keystroke — ⏎ opens a task and runs the start dialog
// — the sentence names both rather than the switch guessing which screen the
// reader is on: the question was about the key.
func (m Model) meaning(k fmt.Stringer) string {
	p := m.opts.Words

	switch {
	case key.Matches(k, m.keys.Open):
		return p.T("tip.open", "opens the task under the cursor; on a band's heading it folds that band away instead")
	case key.Matches(k, m.keys.Start):
		return p.T("tip.start", "opens the dialog that decides what the run will be — flow, engine, model — and starts it")
	case key.Matches(k, m.keys.Compose):
		return p.T("tip.compose", "writes a new task down; nothing runs until you start it")
	case key.Matches(k, m.keys.Menu):
		return p.T("tip.menu", "every verb for the thing under the cursor, including the ones it refuses and why")
	case key.Matches(k, m.keys.Pause):
		return p.T("tip.pause", "asks a run to stop at its next phase; the phase it is inside finishes first")
	case key.Matches(k, m.keys.Resume):
		return p.T("tip.resume", "lets a stopped run carry on, beginning with the phase it is waiting in front of")
	case key.Matches(k, m.keys.Skip):
		return p.T("tip.skip", "lets a stopped run carry on without the phase it is waiting in front of")
	case key.Matches(k, m.keys.Cancel):
		return p.T("tip.cancel", "stops the run for good; the worktree and everything written in it stay where they are")
	case key.Matches(k, m.keys.Requeue):
		return p.T("tip.requeue", "stops the run and puts the task back in to do, to be started again from the top")
	case key.Matches(k, m.keys.Take):
		return p.T("tip.take", "hands you the engine's session in this task's worktree, to type at yourself")
	case key.Matches(k, m.keys.Hand):
		return p.T("tip.hand", "gives the task back to orbit, and the run carries on from where you left it")
	case key.Matches(k, m.keys.Ask):
		return p.T("tip.ask", "writes a note for this task, which the run reads at its next phase")
	case key.Matches(k, m.keys.MarkRead):
		return p.T("tip.read", "marks a finished task read, which is what takes it out of needs you")
	case key.Matches(k, m.keys.Delete):
		return p.T("tip.delete", "removes the task and its whole record; nothing brings either back")
	case key.Matches(k, m.keys.Edit):
		return p.T("tip.edit", "opens this task's own files in $EDITOR")
	case key.Matches(k, m.keys.CLI):
		return p.T("tip.cli", "opens an engine session in this task's worktree, in this terminal, with you at the keyboard")
	case key.Matches(k, m.keys.Filter):
		return p.T("tip.filter", "narrows the board to the tasks whose id, title or repository match what you type")
	case key.Matches(k, m.keys.Commands):
		return p.T("tip.commands", "the command line: everything orbit can do, including what no key was given to")
	case key.Matches(k, m.keys.Autopilot):
		return p.T("tip.autopilot", "turns autopilot on and off; with it on, tasks in to do start by themselves")
	case key.Matches(k, m.keys.Supervisor):
		return p.T("tip.supervisor", "opens the supervisor: what it has said about the board, and the line you say things to it on")
	case key.Matches(k, m.keys.Repos):
		return p.T("tip.repos", "the repositories orbit knows about, and which of them the board is showing")
	case key.Matches(k, m.keys.Knowledge):
		return p.T("tip.knowledge", "what orbit has learned about this code: the rules that stop a run, and what it is told before one")
	case key.Matches(k, m.keys.Flows):
		return p.T("tip.flows", "the flows a run can be made of: which phases it has, what each one is asked, and where it stops for you")
	case key.Matches(k, m.keys.EngineKnobs):
		return p.T("tip.engines", "which engine and model the next run is asked with, and how hard it is asked to think")
	case key.Matches(k, m.keys.Quota):
		return p.T("tip.quota", "what each engine has left of its window, and when the next one opens")
	case key.Matches(k, m.keys.Language):
		return p.T("tip.language", "switches every word on this screen between English and Spanish")
	case key.Matches(k, m.keys.Help):
		return p.T("tip.help", "the whole cheat sheet, every key at once")
	case key.Matches(k, m.keys.Quit):
		return p.T("tip.quit", "closes the cockpit; the runs it started carry on without it")
	case key.Matches(k, m.keys.Back):
		return p.T("tip.back", "goes back one screen, and on the board clears whatever is filtering it")
	}

	// The keys that move about answer with the label they are drawn with.
	// A sentence for each would be the same sentence — it moves the cursor
	// — and the label already says where to.
	for _, b := range []key.Binding{
		m.keys.Up, m.keys.Down, m.keys.First, m.keys.Last, m.keys.PageUp, m.keys.PageDown,
		m.keys.NextTab, m.keys.PrevTab, m.keys.Sideways,
	} {
		if key.Matches(k, b) {
			return b.Help().Desc
		}
	}

	return p.T("tip.nothing", "nothing in this window answers {key}", about("key", k.String()))
}
