package ui

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// barTop is the first row of the screen whose last cell is drawn as the
// bar, which is where the rail the pointer aims at actually starts. Read
// off the render so that a hit test which has drifted from it is caught
// rather than confirmed.
func barTop(t *testing.T, m Model) int {
	t.Helper()

	lines := screenRows(m)

	for y := m.frame.Body.Y; y < m.frame.Body.Y+m.frame.Body.H && y < len(lines); y++ {
		cell := []rune(lines[y])
		if len(cell) <= m.frame.Body.W-1 {
			continue
		}

		if last := string(cell[m.frame.Body.W-1]); last == scrollRail || last == scrollThumb {
			return y
		}
	}

	t.Fatal("no row of the body is drawn with a bar in its last column")

	return 0
}

// barRows is how many rows of the screen the bar was drawn down.
func barRows(t *testing.T, m Model) int {
	t.Helper()

	lines := screenRows(m)
	n := 0

	for y := barTop(t, m); y < m.frame.Body.Y+m.frame.Body.H && y < len(lines); y++ {
		cell := []rune(lines[y])
		if len(cell) <= m.frame.Body.W-1 {
			break
		}

		if last := string(cell[m.frame.Body.W-1]); last != scrollRail && last != scrollThumb {
			break
		}

		n++
	}

	return n
}

// barAt is a window whose pane is long enough to have a bar, and the screen
// coordinates of one row of that bar.
func barAt(t *testing.T, row int) (Model, int, int) {
	t.Helper()

	m, _ := openWith(t, "ACME-2662", longLog())
	m = showing(t, m, tabTimeline)

	if _, rows := m.paneBand(); rows <= 0 {
		t.Fatalf("the pane was given %d rows", rows)
	}

	return m, m.frame.Body.W - 1, barTop(t, m) + row
}

// lineAt is the first line of the pane on show. A viewport in a map cannot
// be asked directly: YOffset has a pointer receiver and a map entry has no
// address.
func lineAt(m Model) int {
	vp := m.panes[m.tab]

	return vp.YOffset()
}

// pointed is the model a mouse message leaves behind.
func pointed(t *testing.T, m Model, e tea.MouseMsg) Model {
	t.Helper()

	next, _ := m.mouse(e)

	return asModel(t, next)
}

// heightOf is the rows the pane of one tab was sized to.
func heightOf(m Model, t tab) int {
	vp := m.panes[t]

	return vp.Height()
}

// scrolledTo puts the pane on show at line n.
func scrolledTo(m Model, n int) Model {
	vp := m.panes[m.tab]
	vp.SetYOffset(n)
	m.panes[m.tab] = vp

	return m
}

// TestTheLastColumnOfALongPaneIsTheBar. The bar is drawn over the pane's
// own last column, so what is under the pointer there is the bar and not the
// line it was drawn on top of — and the row it answers with is the row it
// was drawn on, counted from where the pane starts rather than from the top
// of the window.
func TestTheLastColumnOfALongPaneIsTheBar(t *testing.T) {
	m, x, y := barAt(t, 3)

	got := m.hit(x, y)
	if got.Kind != TargetScrollBar || got.Pane != 3 {
		t.Errorf("the fourth row of the bar = %+v, want the bar's row 3", got)
	}

	if got := m.hit(x-1, y); got.Kind == TargetScrollBar {
		t.Error("the column beside the bar answers as the bar")
	}

	if got := m.hit(x, y-4); got.Kind == TargetScrollBar {
		t.Errorf("the row above the bar answers as the bar: %+v", got)
	}

	// The rail runs the height of the pane, and the pointer answers for
	// every row of it: a hit test that counted more rows than were drawn
	// would read the whole document short.
	if _, rows := m.paneBand(); barRows(t, m) != rows {
		t.Errorf("the bar is drawn down %d rows and the pane is %d", barRows(t, m), rows)
	}
}

// TestAPaneWithNothingToScrollHasNoBarToPointAt. Nothing is drawn in that
// column when everything fits, and a rail that is not there would take a
// cell of every short pane away from the pane.
func TestAPaneWithNothingToScrollHasNoBarToPointAt(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", longLog())
	m = showing(t, m, tabTimeline)
	m = asModel(t, mustUpdate(m, tea.WindowSizeMsg{Width: 120, Height: 200}))

	if m.barShows() {
		t.Fatal("a window taller than the log still draws a bar")
	}

	// A row under the log rather than one of its own: the rules and the
	// entries that fold answer for themselves across their whole width, and
	// what is being asked here is what the column holds when nothing is
	// drawn in it.
	top, rows := m.paneBand()

	below := m.panes[tabTimeline].TotalLineCount()
	if top+below >= rows {
		t.Fatalf("the log fills the pane at %d rows, so no row of it is blank", rows)
	}

	if got := m.hit(m.frame.Body.W-1, m.frame.Body.Y+top+below); got.Kind != TargetPaneBody {
		t.Errorf("the last column of a pane with nothing to scroll = %+v, want the pane", got)
	}
}

// TestPointingAtTheRailGoesThere. The whole of what a bar is for: the top of
// it is the top of the text, the floor of it is the end.
func TestPointingAtTheRailGoesThere(t *testing.T) {
	m, _, _ := barAt(t, 0)
	_, rows := m.paneBand()

	vp := m.panes[tabTimeline]
	total := vp.TotalLineCount()

	deep := m.scrollTo(rows - 1)
	if got := lineAt(deep); got != total-rows {
		t.Errorf("the floor of the rail left the view at line %d of %d, want %d", got, total, total-rows)
	}

	if got := lineAt(deep.scrollTo(0)); got != 0 {
		t.Errorf("the top of the rail left the view at line %d", got)
	}
}

// TestTheBarMovesWhileItIsHeld. A press takes hold of the bar and acts at
// once; every cell the pointer crosses while it is down moves the view. A
// bar that waited for the button to come up would be a bar nobody could
// aim, because the row a drag ends on is not the row it started on.
func TestTheBarMovesWhileItIsHeld(t *testing.T) {
	m, x, y := barAt(t, 1)

	vp := m.panes[tabTimeline]
	total := vp.TotalLineCount()
	_, rows := m.paneBand()

	held := pointed(t, m, tea.MouseClickMsg{Button: tea.MouseLeft, X: x, Y: y})

	if held.held.target.Kind != TargetScrollBar {
		t.Fatalf("the press did not take hold of the bar: %+v", held.held.target)
	}

	if got, want := lineAt(held), 1*total/rows; got != want {
		t.Errorf("the press left the view at line %d, want %d", got, want)
	}

	down := rows / 2
	dragged := pointed(t, held, tea.MouseMotionMsg{Button: tea.MouseLeft, X: x, Y: y + down - 1})

	if got, want := lineAt(dragged), down*total/rows; got != want {
		t.Errorf("a drag to row %d left the view at line %d, want %d", down, got, want)
	}

	// Off the rail sideways, with the button still down: a drag is read as
	// the row it is on, and the pointer leaving the column does not drop it.
	back := pointed(t, dragged, tea.MouseMotionMsg{Button: tea.MouseLeft, X: 4, Y: y})

	if got, want := lineAt(back), 1*total/rows; got != want {
		t.Errorf("a drag that left the rail is at line %d, want %d", got, want)
	}

	// Past either end of the rail the view stops at that end, because a
	// pointer dragged off the bottom of a window is still asking for the
	// bottom of the document.
	under := pointed(t, back, tea.MouseMotionMsg{Button: tea.MouseLeft, X: x, Y: m.frame.Body.Y + m.frame.Body.H - 1})
	if got, want := lineAt(under), total-rows; got != want {
		t.Errorf("a drag under the rail left the view at line %d, want %d", got, want)
	}

	over := pointed(t, under, tea.MouseMotionMsg{Button: tea.MouseLeft, X: x, Y: m.frame.Body.Y})
	if got := lineAt(over); got != 0 {
		t.Errorf("a drag over the top of the rail left the view at line %d", got)
	}
}

// TestNothingIsHeldWithoutAPress. Motion with the button up is a pointer
// crossing the window, and a view that scrolled under it would be a window
// that moves when nobody is touching it.
func TestNothingIsHeldWithoutAPress(t *testing.T) {
	m, x, y := barAt(t, 1)

	loose := pointed(t, m, tea.MouseMotionMsg{X: x, Y: y + 4})
	if got := lineAt(loose); got != lineAt(m) {
		t.Errorf("the view moved to line %d under a pointer nobody was pressing", got)
	}
}

// TestTheWheelTurnsOverTheWholePane. The pane grew two things a pointer can
// be over that are not its text — the heads that fold a section and the bar
// down its edge — and the wheel has to keep turning over both of them.
func TestTheWheelTurnsOverTheWholePane(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", longLog())
	m = scrolledTo(showing(t, m, tabOverview), 4)

	top := barTop(t, m) - m.frame.Body.Y

	head := -1
	for doc := range m.overviewFoldRows() {
		if doc >= 4 && (head < 0 || doc < head) {
			head = doc
		}
	}

	if head < 0 {
		t.Fatal("the overview has no section head under the top of the pane")
	}

	for _, c := range []struct {
		what string
		x, y int
	}{
		{"a section head", 4, m.frame.Body.Y + top + head - 4},
		{"the bar", m.frame.Body.W - 1, m.frame.Body.Y + top + 1},
	} {
		got := m.wheel(tea.Mouse{Button: tea.MouseWheelUp, X: c.x, Y: c.y})
		if lineAt(got) >= 4 {
			t.Errorf("the wheel over %s left the view at line %d", c.what, lineAt(got))
		}
	}
}

// TestTheFileSelectorIsNotTheBar. The diff's selector is drawn between the
// tab strip and the pane, so the bar starts under it. A rail that began at
// the tab strip would answer for rows of the selector, and would read every
// row of the pane one document line too far down.
func TestTheFileSelectorIsNotTheBar(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", longLog())
	m = showing(t, m, tabDiff)

	var diff strings.Builder
	for _, f := range []string{"internal/ui/wheel.go", "internal/ui/mouse.go"} {
		diff.WriteString("diff --git a/" + f + " b/" + f + "\n--- a/" + f + "\n+++ b/" + f + "\n@@ -1,20 +1,20 @@\n")

		for i := range 20 {
			diff.WriteString("+\tline " + strconv.Itoa(i) + "\n")
		}
	}

	m.diff = diff.String()
	m.diffKnown = true
	m.diffFilePicker = true
	m = m.syncPanes()

	top, rows := m.paneBand()
	if !m.barShows() {
		t.Fatal("a diff of forty lines draws no bar")
	}

	// The selector's rows come out of the pane's, so a pane sized as though
	// the selector were not there ends its last page below the window: the
	// bar's floor would point at lines nobody can bring into view.
	if got := heightOf(m, tabDiff); got != rows {
		t.Errorf("the diff pane is %d rows tall and is drawn into %d", got, rows)
	}

	bodyStart := len(m.detailHeadLines(m.frame.Body.W)) + 2
	if top <= bodyStart {
		t.Fatalf("the selector took no rows: the pane starts at %d, under a strip that ends at %d", top, bodyStart)
	}

	for line := bodyStart; line < top; line++ {
		if got := m.hit(m.frame.Body.W-1, m.frame.Body.Y+line); got.Kind == TargetScrollBar {
			t.Errorf("row %d of the selector answers as the bar: %+v", line-bodyStart, got)
		}
	}
}
