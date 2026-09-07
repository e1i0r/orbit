package point

// What one cell of the terminal holds, so that a click can be answered.
//
// It is a package of its own because every screen writes these and the
// window reads them: a screen that has become a package still has to be able
// to say "a click here means this flow", and the alternative is each screen
// inventing its own vocabulary for the same thing and the window translating
// between them.

import (
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/view"
)

// Kind is what sort of thing a cell holds.
type Kind int

// The kinds of thing a cell can hold.
const (
	None Kind = iota
	Task
	BandHeader
	BarHint
	HeaderField
	StatusField
	HeaderQueue
	SettingsRow
	EngineRow
	PaneTab
	PaneBody
	DialogPhase
	DialogSwitch
	Command
	MenuEntry
	Repo
	FlowItem
	ComposeField
	ComposeCaret
	ComposeTab
	ComposeFlowChoice
	ComposeNewFlow
	ComposeInspectFlow
	ComposeAction
	ComposePaste
	DiffFile
	DiffSelectToggle
	Fold
	Seam
	PaneRow
	ScrollBar
)

// Target is what one cell holds.
type Target struct {
	Kind   Kind
	ID     string        // Task, Repo
	Band   view.Band     // Task, BandHeader
	Column layout.Column // Task: which field of the row was pointed at
	// PaneTab, PaneBody, SettingsRow, ScrollBar: the
	// row of the bar. PaneRow: which entry. Seam: which attempt.
	Pane  int
	Key   string // BarHint, MenuEntry, Command, Fold
	Field string // HeaderField, StatusField
	// DialogPhase: which phase. ComposeCaret: which drawn line
	// of the box, counted from the first one on screen.
	Phase int
	// Caret is the column of the drawn line that was pointed at, for
	// ComposeCaret. A field is a place somebody points inside, not
	// only one they land on.
	Caret int
}

// Same is whether two targets are the same thing, disregarding which field
// of it was pointed at.
func (t Target) Same(o Target) bool {
	t.Column, o.Column = 0, 0
	return t == o
}
