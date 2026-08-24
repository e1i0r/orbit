package ui

// What is under one cell of the terminal. A single question — "what did the
// reader point at?" — answered from the same facts View draws from, and from
// nothing else.
//
// It is a file of its own, and a pure function, because of the shape the
// alternative takes. The program this replaces answered this question inside
// its mouse handler, from offsets written a second time next to the ones the
// renderer used; the two drifted the first time a region changed height, and
// a click then landed one row above the row the reader had aimed at. Here
// there is one set of numbers — the frame's and the column plan's — and both
// the drawing and the pointing read it.

import (
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/view"
)

// TargetKind is what sort of thing a cell holds.
//
// TargetNone is the zero value and means nothing actionable is there: a
// blank line, the gap between two columns, a rule, a cell past the end of
// the frame. A caller that ignores the kind acts on nothing rather than on
// whatever happened to be first.
type TargetKind int

const (
	TargetNone TargetKind = iota
	TargetTask
	TargetBandHeader
	TargetBarHint
	TargetHeaderField
	TargetStatusField
	TargetPaneTab
	TargetPaneBody
	TargetDialogPhase
	TargetDialogSwitch
	TargetCommand
	TargetMenuEntry
	TargetRepo
)

// Target is one cell's answer: what kind of thing is there, and enough to
// act on it without asking the geometry a second time.
//
// It is a struct rather than a bare kind because acting needs the thing
// itself — a task's id, a band's name, a pane's number — and a handler
// handed only a kind would have to go back to the rows and the offsets to
// find out which one, which is the re-derivation this file exists to remove.
//
// Only the fields its Kind names are set; every other one is zero, and Band
// in particular is a real band at zero — so read it when the kind says to
// and never to ask whether it was set.
type Target struct {
	Kind TargetKind

	ID     string        // TargetTask, TargetRepo
	Band   view.Band     // TargetTask, TargetBandHeader
	Column layout.Column // TargetTask: which field of the row was pointed at
	Pane   int           // TargetPaneTab, TargetPaneBody
	Key    string        // TargetBarHint, TargetMenuEntry, TargetCommand
	Field  string        // TargetHeaderField, TargetStatusField
	Phase  int           // TargetDialogPhase
}

// hit is what is at cell (x, y), counted from the top left of the terminal.
//
// The routing is by screen first and by region second, in that order,
// because that is the order View builds a frame in: the header, the band and
// the bar are drawn the same way on all three screens, and only the body
// changes with the screen. An arm that has nothing to answer yet answers
// TargetNone rather than guessing, so a cell whose meaning has not been
// decided does nothing when it is clicked — which is the one behaviour a
// reader can safely be wrong about.
func (m Model) hit(x, y int) Target {
	if m.width <= 0 || m.height <= 0 || m.tooNarrow {
		// A refusal is one sentence and two numbers, and there is nothing
		// on it to point at.
		return Target{}
	}
	switch m.screen {
	case screenDetail:
		return m.hitDetail(x, y)
	case screenStart:
		return m.hitStart(x, y)
	}
	return m.hitBoard(x, y)
}

// hitBoard is the list screen.
func (m Model) hitBoard(x, y int) Target {
	switch m.frame.At(y) {
	case layout.RegionBody:
		return m.hitRow(x, y)
	case layout.RegionHeader:
		// The header's standing fields — the unread pair, the autopilot
		// pip, the repository count — are each a switch a reader will
		// expect to be able to click. Which cells each one occupies is
		// decided where they are laid out, and that is not settled until
		// the header carries a repository list.
		return Target{}
	case layout.RegionBar:
		return m.hitBar(x, y)
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
	_, hints := m.barLayout(m.frame.Bar.W)
	for _, h := range hints {
		if x >= h.x && x < h.x+h.w {
			return Target{Kind: TargetBarHint, Key: h.key}
		}
	}
	return Target{}
}

// hitDetail is the task view, one level down: its tabs, the pane under them,
// and the key bar. Nothing is pointed at there yet.
func (m Model) hitDetail(x, y int) Target { return Target{} }

// hitStart is the dialog that decides what a run will be: its phases, its
// switches, and the flow along the top. Nothing is pointed at there yet.
func (m Model) hitStart(x, y int) Target { return Target{} }
