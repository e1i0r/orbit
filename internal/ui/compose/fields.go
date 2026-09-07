package compose

// The rows of the form, in the order the cursor walks them, and which of
// them is being typed into.

import "github.com/e1i0r/orbit/internal/ui/typing"

// The fields of the form. Which engine, which model, how much thinking and
// how much effort are not among them: the seat already answers those, and a
// phase of a flow answers them again for the work it runs. A form that asks
// a third time is a third answer to keep in sync with the other two.
//
// Neither is the repository, and that one was the first field on the screen.
// Choosing one was a reader's first decision about a task, back when a task
// belonged to the checkout it was written under. It is worked in whatever it
// turns out to reach into now, so the field asked a question the answer to
// which was no longer the reader's to give — and a task that reached into
// three repositories still had one of them declared at the top of it as if
// that were the fact about it.
const (
	composeFlow = iota
	composeID
	composeText
	composeFields
)

// The URL tab, in the order the rows are drawn and the order the cursor
// walks them. The URL is last because it is what this tab is for: the
// reader pastes it, looks at the flow above to see what will be done with
// it, and saves — rather than reading upwards from the thing they came for.
const (
	composeURLFlow = iota
	composeURL
	composeURLFields
)

// active is the field being typed into, or nothing when the form is on a
// row of pills. Every key that writes, deletes or moves a caret goes
// through it, so which field a keystroke lands in is answered once.
func (s *State) active() *typing.Field {
	if s.tab == composeTabURL {
		if s.field == composeURL {
			return &s.url
		}

		return nil
	}

	switch s.field {
	case composeID:
		return &s.id
	case composeText:
		return &s.text
	}

	return nil
}

// typed is what the field being typed into holds, for the screens that only
// want to read it.
func (s *State) typed() string {
	if in := s.active(); in != nil {
		return in.String()
	}

	return ""
}

func (s State) composeMove(d int) State {
	maxFields := composeFields
	if s.tab == composeTabURL {
		maxFields = composeURLFields
	}

	s.field += d
	if s.field < 0 {
		s.field = 0
	}

	if s.field >= maxFields {
		s.field = maxFields - 1
	}

	return s
}

func (s State) composeTab(d int) State {
	return s.composeMove(d)
}

func (s State) composeNext(startNow bool, e Env) (State, Out) {
	limit := composeText
	if s.tab == composeTabURL {
		limit = composeURL
	}

	if s.field < limit {
		s.field++
		return s, Out{}
	}

	return s.Submit(startNow, e)
}

// firstComposeField is where the cursor lands when a tab is opened: the
// field that tab exists for, which on the URL tab is the last row and not
// the first.
func firstComposeField(tab int) int {
	if tab == composeTabURL {
		return composeURL
	}

	return composeFlow
}
