package ui

// What a key does on a task's screen, where detailKey takes a letter before
// the key map does: p there opens a pull request and does not pause, E
// turns the effort rather than opening the engines, k opens them rather
// than moving up. ? and a hover answered with the board's sentence, which
// described a key the screen had given another job.
//
// The order is detailKey's, so the two are read side by side.

import (
	"fmt"
)

// meaningHere is the sentence for a key where the reader is standing: the
// task screen's own when it has one, the key map's otherwise.
//
// It is for ? and the hover only. The menu and the cheat sheet ask about a
// binding and not a keystroke, and a binding means what the key map says.
func (m Model) meaningHere(k fmt.Stringer) string {
	if m.screen == screenDetail {
		if says, ok := m.taskScreenMeaning(k.String()); ok {
			return says
		}
	}

	return m.meaning(k)
}

// taskScreenMeaning is the sentence for a letter the task screen takes for
// a job of its own, on the tab in front of the reader.
func (m Model) taskScreenMeaning(k string) (string, bool) {
	p := m.opts.Words

	switch {
	case m.tab == tabDiff && k == "}":
		return p.T("tip.detail.next_hunk", "jumps to the next change in the diff"), true
	case m.tab == tabDiff && k == "{":
		return p.T("tip.detail.prev_hunk", "jumps to the change before this one in the diff"), true
	case m.tab == tabDiff && k == "f":
		return p.T("tip.detail.file_picker", "lists the files of the diff, to jump to one"), true
	case m.tab == tabDiff && (k == "z" || k == " " || k == "space"):
		return p.T("tip.detail.fold_file", "folds the file under the cursor away, or opens it again"), true
	case m.tab == tabOverview && (k == "z" || k == "Z"):
		return p.T("tip.detail.fold_all", "folds every section of the overview to its heading, or opens them all"), true
	case m.tab == tabDiff && k == "Z":
		return p.T("tip.detail.fold_every_file", "folds every file of the diff away, or opens them all"), true
	case m.tab == tabImpact && k == "B":
		return p.T("tip.detail.compare", "runs the flow's own checks on both sides of the change, before and after it"), true
	case m.tab == tabDiff && k == "H":
		return p.T("tip.detail.rationale", "shows or hides the engine's reasons beside the lines it wrote"), true
	}

	switch k {
	case "J":
		return p.T("tip.detail.merge_pr", "merges this task's pull request and removes its branch"), true
	case "U":
		return p.T("tip.detail.update_pr", "asks the supervisor to bring the pull request up to date with its base branch"), true
	case "X":
		return p.T("tip.detail.close_pr", "closes this task's pull request, after asking why"), true
	case "e":
		return p.T("tip.detail.expand", "switches the overview between one line per field and every field in full"), true
	case "v", "V":
		return p.T("tip.detail.raw", "switches between the text formatted and the text as it was written"), true
	case "p", "P":
		return p.T("tip.detail.create_pr", "opens a pull request for this task's branch"), true
	case "C":
		return p.T("tip.detail.fix_checks", "asks the supervisor to fix the checks that failed on the pull request"), true
	case "T":
		return p.T("tip.detail.more_tests", "asks the supervisor for the tests this change is missing"), true
	case "R":
		return p.T("tip.detail.resolve", "asks the supervisor to answer the review comments on the pull request"), true
	case "D":
		return p.T("tip.detail.review", "a deep review of the pull request, written on it; nothing is changed"), true
	case "t":
		return p.T("tip.detail.thinking", "turns the thinking mode the next run is asked with"), true
	case "k", "K":
		return p.T("tip.engines", "which engine and model the next run is asked with, and how hard it is asked to think"), true
	case "E":
		return p.T("tip.detail.effort", "turns the effort the next run is asked to reason with"), true
	}

	return "", false
}
