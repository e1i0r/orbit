package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// foldLabels is what each section is headed with in English, which is the
// language every window in these tests is drawn in.
var foldLabels = map[string]string{
	foldPhases:  "FLOW",
	foldChanges: "CHANGES",
	foldDeliver: "DELIVER",
}

// screenRows is the whole window as it is drawn, one string per terminal
// row, so a test can point at the row a thing landed on.
func screenRows(m Model) []string {
	return strings.Split(ansi.Strip(m.View().Content), "\n")
}

// rowOf is the first row of lines that mentions want, or -1.
func rowOf(lines []string, want string) int {
	for i, l := range lines {
		if strings.Contains(l, want) {
			return i
		}
	}

	return -1
}

// TestASectionSaysWhetherItFolds. The arrow is the whole of the affordance:
// a head drawn without one is read as a label, and a reader who cannot see
// that a block folds never folds one.
func TestASectionSaysWhetherItFolds(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", fixtureEntries())

	for key, label := range foldLabels {
		if got := overviewText(m); !strings.Contains(got, foldOpen+label) {
			t.Errorf("the open %s section has no arrow on it:\n%s", label, got)
		}

		if got := overviewText(m.fold(key)); !strings.Contains(got, foldShut+label) {
			t.Errorf("the closed %s section has no arrow on it:\n%s", label, got)
		}
	}
}

// TestAFoldedSectionKeepsOnlyItsHead. Folding is for the reader who wants
// the screen back: what it holds goes, what it is stays, and the count it
// leaves behind is so a fold hides a section's detail and not its existence.
func TestAFoldedSectionKeepsOnlyItsHead(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", fixtureEntries())

	before := overviewText(m)
	if !strings.Contains(before, "press [2] for full flow tree") {
		t.Fatalf("the open pane does not carry flow card hint:\n%s", before)
	}

	after := overviewText(m.fold(foldPhases))

	if strings.Contains(after, "press [2] for full flow tree") {
		t.Errorf("a folded section still sets what it holds:\n%s", after)
	}

	if !strings.Contains(after, foldShut+"FLOW") {
		t.Errorf("a folded section does not show flow name:\n%s", after)
	}

	if strings.Contains(before, foldShut+"FLOW") {
		t.Errorf("an open section is folded:\n%s", before)
	}

	if len(strings.Split(after, "\n")) >= len(strings.Split(before, "\n")) {
		t.Error("folding a section did not shorten the pane")
	}
}

// TestTheHeadsAreWhereTheHitTestSaysTheyAre. A click is answered by counting
// the rows the blocks drew and the pane is drawn by joining those blocks, so
// the two agreeing is the whole of whether clicking a head folds the section
// under the pointer rather than the one above it.
func TestTheHeadsAreWhereTheHitTestSaysTheyAre(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", fixtureEntries())
	lines := strings.Split(overviewText(m), "\n")

	rows := m.overviewFoldRows()
	if len(rows) != len(foldLabels) {
		t.Fatalf("the hit test knows %d heads, want %d", len(rows), len(foldLabels))
	}

	for row, key := range rows {
		if row < 0 || row >= len(lines) {
			t.Fatalf("the hit test puts %s on row %d of a pane %d rows tall", key, row, len(lines))
		}

		if want := foldOpen + foldLabels[key]; !strings.Contains(lines[row], want) {
			t.Errorf("the hit test puts %s on row %d, which is %q", key, row, lines[row])
		}
	}
}

// TestAHeadIsClickableWhereItIsDrawn. The rows the hit test counts are pane
// rows and a pointer reports terminal rows, so the head has to answer at the
// cell it was drawn in, under the header and the tab strip above it.
func TestAHeadIsClickableWhereItIsDrawn(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", fixtureEntries())

	// Tall enough for every head to be on the screen at once: a head the
	// reader would have to scroll to is a head this test cannot point at.
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 60})
	m = asModel(t, next)

	lines := screenRows(m)

	for key, label := range foldLabels {
		y := rowOf(lines, foldOpen+label)
		if y < 0 {
			t.Fatalf("the %s head was not drawn on the window:\n%s", label, strings.Join(lines, "\n"))
		}

		got := m.hit(4, y)
		if got.Kind != TargetFold || got.Key != key {
			t.Errorf("a click on the %s head at row %d = %+v, want the %s fold", label, y, got, key)
		}
	}
}

// TestClickingAHeadFoldsThatSection. The gesture, end to end: the target the
// hit test made is the one the click acts on.
func TestClickingAHeadFoldsThatSection(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", fixtureEntries())

	next, _ := m.leftClick(Target{Kind: TargetFold, Key: foldChanges})
	m = asModel(t, next)

	if !m.folded(foldChanges) {
		t.Error("clicking the changes head left it open")
	}

	next, _ = m.leftClick(Target{Kind: TargetFold, Key: foldChanges})
	if m = asModel(t, next); m.folded(foldChanges) {
		t.Error("clicking a folded head left it closed")
	}
}

// TestOneKeyFoldsEverySection. The pointer folds them one at a time; a
// reader on the keyboard gets the screen back in one keystroke, and the same
// keystroke gives it all back.
func TestOneKeyFoldsEverySection(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", fixtureEntries())

	next, _ := m.detailKey(keystroke("z"))
	m = asModel(t, next)

	for _, key := range overviewFolds {
		if !m.folded(key) {
			t.Errorf("z left the %s section open", key)
		}
	}

	next, _ = m.detailKey(keystroke("z"))
	m = asModel(t, next)

	for _, key := range overviewFolds {
		if m.folded(key) {
			t.Errorf("z on a folded pane left the %s section closed", key)
		}
	}
}

// TestAHeadIsStillItselfAfterAScroll. The hit test counts rows of the pane's
// content and the pointer reports rows of the terminal, so whatever the
// reader has scrolled past stands between the two.
func TestAHeadIsStillItselfAfterAScroll(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", fixtureEntries())
	m.panes[tabOverview].SetYOffset(6)

	lines := screenRows(m)

	y := rowOf(lines, foldOpen+"FLOW")
	if y < 0 {
		t.Fatalf("the flow head was not drawn after a scroll:\n%s", strings.Join(lines, "\n"))
	}

	if got := m.hit(4, y); got.Kind != TargetFold || got.Key != foldPhases {
		t.Errorf("a click on the scrolled flow head = %+v, want the flow fold", got)
	}
}

// TestOnlyTheOverviewHasHeads. Every pane is drawn in the same region, so a
// row number that is a head on one of them is an ordinary line on the other
// ten — and folding a section nobody can see is a screen that changes behind
// the reader's back.
func TestOnlyTheOverviewHasHeads(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", fixtureEntries())
	m = showing(t, m, tabTimeline)

	for row := range m.overviewFoldRows() {
		if key, ok := m.hitFold(row); ok {
			t.Errorf("row %d of the timeline answers the %s fold", row, key)
		}
	}
}
