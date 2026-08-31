package ui

// The compose form under the pointer. Every dial on it is a row of pills,
// and a pill nobody can land on is a choice the form does not really offer:
// these tests reach each one the way a reader does, by the cell it was drawn
// in rather than by a column written into the test.

import "testing"

// formRow is the screen row a labelled line of the form was drawn on.
func formRow(t *testing.T, m Model, label string) int {
	t.Helper()

	y := rowOf(screenRows(m), label)
	if y < 0 {
		t.Fatalf("the form has no row saying %q", label)
	}

	return y
}

// pointAt is the first cell of row y the window answers for with the target
// asked for. A dial whose pill is not reachable fails here rather than in an
// assertion further down, because the two are different faults: one is a
// pill drawn where nothing is listening, the other is a click that did the
// wrong thing.
func pointAt(t *testing.T, m Model, y int, kind TargetKind, pane int) int {
	t.Helper()

	for x := range m.width {
		if at := m.hit(x, y); at.Kind == kind && at.Pane == pane {
			return x
		}
	}

	t.Fatalf("no cell of row %d answers with kind %d, pane %d", y, kind, pane)

	return -1
}

// clickAt presses and releases on the cell pointAt found.
func clickAt(t *testing.T, m Model, y int, kind TargetKind, pane int) Model {
	t.Helper()

	x := pointAt(t, m, y, kind, pane)

	after, _ := m.leftClick(m.hit(x, y))

	return asModel(t, after)
}

// composeForm is the form open on a window wide enough to draw all of it.
func composeForm(t *testing.T) Model {
	t.Helper()

	m, _ := testModel(t, 120, 40)

	return m.openCompose()
}

// TestEveryDialOfTheFormIsChosenByPointingAtIt. Two rows of pills, and the
// pointer is the only way to reach one of them that does not involve
// counting keystrokes.
func TestEveryDialOfTheFormIsChosenByPointingAtIt(t *testing.T) {
	m := composeForm(t)

	m = clickAt(t, m, formRow(t, m, "repository:"), TargetComposeRepoChoice, 1)
	if m.compose.repoIdx != 1 || m.compose.repo != m.compose.repos[1].name {
		t.Errorf("the second repository pill left repoIdx=%d repo=%q, want 1 and %q",
			m.compose.repoIdx, m.compose.repo, m.compose.repos[1].name)
	}

	m = clickAt(t, m, formRow(t, m, "flow:"), TargetComposeFlowChoice, 2)
	if m.compose.flowIdx != 2 {
		t.Errorf("the third flow pill left flowIdx=%d, want 2", m.compose.flowIdx)
	}
}

// TestPointingBesideThePillsPutsTheCursorOnThatRow. The empty half of a dial
// row is still that dial: a reader who lands there has said which field they
// are on, and the keyboard takes it from there.
func TestPointingBesideThePillsPutsTheCursorOnThatRow(t *testing.T) {
	m := composeForm(t)

	y := formRow(t, m, "flow:")

	at := m.hit(m.width-2, y)
	if at.Kind != TargetComposeField || at.Pane != composeFlow {
		t.Fatalf("the far end of the flow row is kind %d pane %d, want the flow field", at.Kind, at.Pane)
	}

	after, _ := m.leftClick(at)
	if m = asModel(t, after); m.compose.field != composeFlow {
		t.Errorf("the cursor is on field %d, want the flow one", m.compose.field)
	}
}

// TestTheFlowRowOffersMoreThanFlows. Two things sit at the end of it — the
// button that writes a new flow, and, one row down, the summary of the flow
// that is chosen. Both open a screen, and neither is a flow.
func TestTheFlowRowOffersMoreThanFlows(t *testing.T) {
	m := composeForm(t)

	y := formRow(t, m, "flow:")

	x := pointAt(t, m, y, TargetComposeNewFlow, 0)

	after, _ := m.leftClick(m.hit(x, y))
	if got := asModel(t, after); got.screen != screenFlows {
		t.Errorf("the New button left screen %v, want the flows screen", got.screen)
	}

	// The summary line under the row: pointing at it inspects the flow the
	// form is set to, which is the same thing clicking the chosen pill does.
	sum := m.hit(20, y+1)
	if sum.Kind != TargetComposeInspectFlow {
		t.Fatalf("the summary line is kind %d, want the flow inspector", sum.Kind)
	}

	after, _ = m.leftClick(sum)
	if got := asModel(t, after); !got.flows.showingDetail {
		t.Error("pointing at the summary did not open the flow it summarises")
	}
}

// TestTheFormsButtonsAreWhereTheyAreDrawn. Save, save and run, and cancel
// are three boxes on one line, and their widths come from the words the
// reader's language spends on them.
func TestTheFormsButtonsAreWhereTheyAreDrawn(t *testing.T) {
	m := composeForm(t)

	y := formRow(t, m, "Save")

	var seen []string

	for x := range m.width {
		at := m.hit(x, y)
		if at.Kind != TargetComposeAction {
			continue
		}

		if len(seen) == 0 || seen[len(seen)-1] != at.Key {
			seen = append(seen, at.Key)
		}
	}

	want := []string{"save", "save_and_run", "cancel"}
	if len(seen) != len(want) {
		t.Fatalf("the actions row answers with %v, want %v in that order", seen, want)
	}

	for i, k := range want {
		if seen[i] != k {
			t.Fatalf("the actions row answers with %v, want %v in that order", seen, want)
		}
	}

	after, _ := m.leftClick(Target{Kind: TargetComposeAction, Key: "cancel"})
	if got := asModel(t, after); got.screen == screenCompose {
		t.Error("cancel left the form open")
	}
}

// TestTheOtherTabHasItsOwnRows. The form is two forms — one a task is typed
// into and one an issue is fetched into — and the second is reached by
// pointing at its tab.
func TestTheOtherTabHasItsOwnRows(t *testing.T) {
	m := composeForm(t)

	tabs := formRow(t, m, "Manual")

	after, _ := m.leftClick(m.hit(m.width/2, tabs))

	m = asModel(t, after)
	if m.compose.tab != composeTabURL {
		t.Fatalf("the right half of the tab line left tab %d, want the URL one", m.compose.tab)
	}

	// The URL row carries the paste button, and the rest of the cells on it
	// are the field itself.
	y := formRow(t, m, "url:")
	if at := m.hit(20, y); at.Kind != TargetComposePaste {
		t.Errorf("the paste button of the url row is kind %d, want the paste target", at.Kind)
	}

	if at := m.hit(m.width-2, y); at.Kind != TargetComposeCaret || at.Pane != composeURL {
		t.Errorf("the far end of the url row is kind %d pane %d, want the url field", at.Kind, at.Pane)
	}

	// Every field of this tab is its own, not the manual tab's.
	m = clickAt(t, m, formRow(t, m, "repository:"), TargetComposeRepoChoice, 1)
	if m.compose.repoIdx != 1 {
		t.Errorf("the second repository pill of the url tab left repoIdx=%d, want 1", m.compose.repoIdx)
	}

	if at := m.hit(m.width-2, formRow(t, m, "flow:")); at.Pane != composeURLFlow {
		t.Errorf("the flow row of the url tab is field %d, want the url one", at.Pane)
	}
}

// TestThePasteButtonIsNotTheTaskField. They share a row, and the one the
// reader lands on decides whether the clipboard is read or the cursor moves.
func TestThePasteButtonIsNotTheTaskField(t *testing.T) {
	m := composeForm(t)

	y := formRow(t, m, "📋 Paste")

	if at := m.hit(20, y); at.Kind != TargetComposePaste {
		t.Errorf("the paste button is kind %d, want the paste target", at.Kind)
	}

	if at := m.hit(m.width-2, y); at.Kind != TargetComposeField || at.Pane != composeText {
		t.Errorf("the far end of the task row is kind %d pane %d, want the text field", at.Kind, at.Pane)
	}

	// The box under it is the same field: a reader aiming at the text aims
	// at the box, not at the word above it.
	if at := m.hit(20, y+3); at.Kind != TargetComposeCaret || at.Pane != composeText {
		t.Errorf("inside the text box is kind %d pane %d, want the text field", at.Kind, at.Pane)
	}
}

// TestTheBlankRowsOfTheFormAnswerForNothing. A form that answered for every
// cell of its body would make the gap under the tabs clickable, and a reader
// who misses a row by one would move the cursor without meaning to.
func TestTheBlankRowsOfTheFormAnswerForNothing(t *testing.T) {
	m := composeForm(t)

	tabs := formRow(t, m, "Manual")
	if at := m.hit(2, tabs+1); at.Kind != TargetNone {
		t.Errorf("the blank line under the tabs answers with kind %d, want nothing", at.Kind)
	}
}
