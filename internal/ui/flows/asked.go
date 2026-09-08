package flows

// What the window asks of the designer without changing it.
//
// They are here rather than beside the state they read because they are one
// subject: the handful of questions the window has to be able to answer
// about a screen it does not own — is a paste going into a form, is a
// question out at an engine, does the wheel scroll a list.

import "time"

// Creating is whether the designer is editing a flow rather than listing
// them, and Saying whether a question is out at an engine. The window asks
// because a paste goes into the form and the spinner turns while a draft is
// out.
func (s State) Creating() bool { return s.creating }

// Previewing is whether one flow is open read-only, which is what the
// compose form sends a reader here for.
func (s State) Previewing() bool { return s.showingDetail }

// OnFields opens the tab where a flow is edited by hand. Creating one opens
// on the tab that describes it in words instead, which is the right first
// move for an empty form and the wrong one for a reader who came to change a
// field.
func (s State) OnFields() State {
	s.tab = flowTabFields

	return s
}

// Editing is the name of the flow being changed, and empty while a new one
// is being written: the window says which of the two the reader is in.
func (s State) Editing() string {
	if !s.isEditing {
		return ""
	}

	return s.flowName
}

// Saying is whether an engine is out answering the third tab.
func (s State) Saying() bool { return s.saying }

// Asking is the designer as it looks while a question is out at an engine.
//
// It is a door because what the window draws while it waits — the spinner,
// the count of seconds, the way to stop waiting — is the window's own line,
// and there has to be a way to put a screen in that state without paying an
// engine to answer.
func Asking(engine string, since time.Time) State {
	return State{saying: true, sayEngine: engine, sayAt: since}
}

// Since is when that question went out, for the count beside the spinner.
func (s State) Since() time.Time { return s.sayAt }

// Listing is whether the reader is on the list rather than in a form or a
// preview, which is what decides whether the wheel scrolls the list.
func (s State) Listing() bool { return !s.creating && !s.showingDetail }

// Scroll moves the list under the wheel.
func (s State) Scroll(d int) State {
	s.scroll = max(s.scroll+d, 0)

	return s
}
