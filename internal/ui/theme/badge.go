package theme

import "fmt"

// FormatLatency paints a latency by how much of it a person would notice:
// fast disappears in green, slow admits itself in yellow, and past half a
// second turns red because that is roughly where waiting starts to feel
// like waiting. It is pure layout — no clock, no terminal, just ms in and
// a styled string out.
func FormatLatency(ms int64) string {
	text := fmt.Sprintf("%dms", ms)
	switch {
	case ms < 100:
		return Paint(OK).Render(text)
	case ms < 500:
		return Paint(Warn).Render(text)
	default:
		return Paint(Bad).Render(text)
	}
}

// ActiveTabBadge formats an active inspector tab badge with high-contrast theme selection styling.
func ActiveTabBadge(tag string) string {
	return Paint(Sel).Bold(true).Render(tag)
}

// InactiveTabBadge formats an inactive inspector tab with subtle muted contrast.
func InactiveTabBadge(tag string) string {
	return Paint(Dim).Render(tag)
}

// The colours a pill is drawn in.
//
// Here and not spelled into the screens that draw them. A hex written at the
// call site is a colour no theme can reach: the window changed from frauddi
// to dracula around a Save button that stayed the same green, and a reader
// looking for that green had nowhere to look but nine files under
// internal/ui. Naming the job rather than the hue is the same rule tokens.go
// keeps for text and paper.
//
// They are one value each rather than one per theme, because that is what
// the screens have always drawn. A theme that wants its own is a table here
// and no change at the call sites.
const (
	// PillInk is the label on a pill, and PillInkLit the one on a pill
	// light enough that white would not be read. PillInkRest is a pill
	// nobody has chosen.
	PillInk     = "#FFFFFF"
	PillInkLit  = "#000000"
	PillInkRest = "#94A3B8"

	// The paper each kind of pill is set on.
	PillChosen  = "#A855F7" // the flow the form has picked
	PillRest    = "#1E293B" // one it has not
	PillCreate  = "#6366F1" // write a new flow, and the badge of a written one
	PillPaste   = "#0369A1" // paste, in the form
	PillEdit    = "#0C4A6E" // paste, add a phase, edit — in the designer
	PillDraft   = "#581C87" // ask an engine for the first draft
	PillClear   = "#374151" // throw away what is written
	PillReturn  = "#2563EB" // go back carrying the answer
	PillDelete  = "#7F1D1D" // delete a phase or a flow
	PillSave    = "#14532D" // save a flow
	PillDetails = "#0284C7" // details, and the badge of a built-in flow
	PillSelect  = "#16A34A" // select and return
	PillDesign  = "#4F46E5" // open the designer
	PillBack    = "#334155" // leave
	PillNew     = "#005F87" // create a custom flow
	PillHeader  = "#0F766E" // the window's own name badge
)

// The line around a box the reader types into: at rest, and while the caret
// is in it.
const (
	BoxLine       = "#334155"
	BoxLineActive = "#38BDF8"
)
