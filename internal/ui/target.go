package ui

import (
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/view"
)

// TargetKind is what sort of thing a cell holds.
type TargetKind int

// TargetKind values specify interactive screen target kinds.
const (
	TargetNone TargetKind = iota
	TargetTask
	TargetBandHeader
	TargetBarHint
	TargetHeaderField
	TargetStatusField
	TargetHeaderQueue
	TargetSettingsRow
	TargetEngineRow
	TargetPaneTab
	TargetPaneBody
	TargetDialogPhase
	TargetDialogSwitch
	TargetCommand
	TargetMenuEntry
	TargetRepo
	TargetFlowItem
)

// Target is one cell's hit target.
type Target struct {
	Kind   TargetKind
	ID     string        // TargetTask, TargetRepo
	Band   view.Band     // TargetTask, TargetBandHeader
	Column layout.Column // TargetTask: which field of the row was pointed at
	Pane   int           // TargetPaneTab, TargetPaneBody, TargetSettingsRow
	Key    string        // TargetBarHint, TargetMenuEntry, TargetCommand
	Field  string        // TargetHeaderField, TargetStatusField
	Phase  int           // TargetDialogPhase
}

// The names of the two switches on the start dialog. They are constants
// because the same string is written where the target is made and where it
// is acted on, and a typo in either one is a switch that silently stops
// working rather than a build that fails.
const (
	fieldFlow         = "flow"
	fieldAutopilotOn  = "autopilot.on"
	fieldAutopilotOff = "autopilot.off"
)

// hit is what is at cell (x, y), counted from the top left of the terminal.
//
// The routing is by region first and by screen second, and that order is the
// frame's own: the header, the activity band and the key bar are drawn the
// same way whichever screen is up, and only the body changes with it. Asking
// the screen first would put three copies of the key bar's answer in three
// arms, and the third one would be the one that got forgotten.
//
// An arm with nothing to answer yet answers TargetNone rather than guessing,
// so a cell whose meaning has not been decided does nothing when it is
// clicked — the one behaviour a reader can safely be wrong about.
func (m Model) hit(x, y int) Target {
	if m.width <= 0 || m.height <= 0 || m.tooNarrow {
		// A refusal is one sentence and two numbers, and there is nothing
		// on it to point at.
		return Target{}
	}
	switch m.frame.At(y) {
	case layout.RegionBar:
		if m.palette.open || m.menu.open {
			// The palette's line replaces the bar while it is up, and a
			// menu up means the keyboard it names verbs with is spoken
			// for. A line being typed into has no fields worth pointing
			// at yet; when either grows one, this is where its answer
			// starts.
			return Target{}
		}
		return m.hitBar(x, y)
	case layout.RegionHeader:
		return m.hitHeader(x, y)
	case layout.RegionStatus, layout.RegionBand:
		return m.hitStatus(x, y)
	case layout.RegionBody:
		if m.palette.open {
			return m.hitPalette(x, y)
		}
		if m.menu.open {
			return m.hitMenu(x, y)
		}
		if m.watchUp {
			return Target{}
		}
		switch m.screen {
		case screenDetail:
			return m.hitDetail(x, y)
		case screenStart:
			return m.hitStart(x, y)
		case screenSettings:
			return m.hitSettings(x, y)
		case screenEngines:
			return m.hitEngines(x, y)
		case screenFlows:
			return m.hitFlows(x, y)
		case screenRepos:
			return m.hitRepos(x, y)
		case screenCompose:
			return Target{}
		}
		return m.hitRow(x, y)
	}
	return Target{}
}

// hitRow is one line of the body: a task, a band's heading, or neither.
//
// The three ways a cell in the body can mean nothing are all here, and each
// one is a row the reader can see: the blank line between two bands, the
// "… and N more" line the body spends its last row on, and the space below
// a list shorter than the region.
func (m Model) hitRow(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}
	all := m.rows()
	// page is what the body itself drew with, so the row it gave up to say
	// how many are hidden is out of reach here by the same arithmetic that
	// put it there — rather than by a second subtraction that has to be
	// kept in step with the first.
	if line >= page(m.frame.Body.H, len(all), m.offset) {
		return Target{}
	}
	i := m.offset + line
	if i < 0 || i >= len(all) {
		return Target{}
	}
	switch r := all[i]; {
	case r.blank:
		return Target{}
	case r.head:
		return Target{Kind: TargetBandHeader, Band: r.band}
	default:
		t := Target{Kind: TargetTask, ID: r.task.ID, Band: r.band}
		// The gutter the cursor's mark is drawn in is not part of a row,
		// which is why the plan was made without it and why it comes off
		// again here. A cell in the gutter is still the task's row — the
		// reader aimed at the line — it is simply not one of its fields.
		t.Column, _ = m.plan.ColumnAt(x - gutter)
		return t
	}
}

// hitBar is a hint in the key bar, and which key it names.
//
// The bar hands back where it put each hint rather than being measured
// again here, so what is clickable is exactly what was drawn — including
// the hints it dropped for want of width, which are not clickable because
// they are not there.
func (m Model) hitBar(x, y int) Target {
	if y != m.frame.Bar.Y {
		// The bar is one row. A region taller than its content has blank
		// rows under it, and nothing is on them.
		return Target{}
	}
	// Right side execution chips in footer: [autopilot] [cli]
	if m.screen == screenList {
		if x >= m.width-28 {
			return Target{Kind: TargetBarHint, Key: "c"}
		}
		if x >= m.width-60 && x < m.width-28 {
			return Target{Kind: TargetStatusField, Field: "autopilot"}
		}
	} else if x >= m.width-32 {
		return Target{Kind: TargetStatusField, Field: "autopilot"}
	}

	_, hints := m.barLayout(m.frame.Bar.W)
	for _, h := range hints {
		if x >= h.x && x < h.x+h.w {
			return Target{Kind: TargetBarHint, Key: h.key}
		}
	}
	return Target{}
}

func (m Model) hitHeader(x, y int) Target {
	if y != m.frame.Header.Y {
		return Target{}
	}
	if x < 10 {
		return Target{Kind: TargetHeaderField, Field: "orbit"}
	}
	if x >= 10 && x < 28 {
		return Target{Kind: TargetHeaderQueue, Band: view.ToDo}
	}
	if x >= 28 && x < 44 {
		return Target{Kind: TargetHeaderQueue, Band: view.Running}
	}
	if x >= 44 && x < 62 {
		return Target{Kind: TargetHeaderQueue, Band: view.NeedsYou}
	}
	if x >= 62 && x < 78 {
		return Target{Kind: TargetHeaderQueue, Band: view.Done}
	}

	// Right side of Header: [repos] [engine] [lang]
	if x >= m.width-12 {
		return Target{Kind: TargetHeaderField, Field: "lang"}
	}
	if x >= m.width-28 && x < m.width-12 {
		return Target{Kind: TargetHeaderField, Field: "engine"}
	}
	if x >= m.width-56 && x < m.width-28 {
		return Target{Kind: TargetHeaderField, Field: "repos"}
	}
	return Target{}
}

func (m Model) hitStatus(x, y int) Target {
	return Target{Kind: TargetStatusField}
}
