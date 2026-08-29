package ui

import (
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// TestClickingAnywhereOnTheBadgeResetsTheFilters.
//
// hitHeader places the badge's click by a number written down in target.go,
// not by measuring what header.go drew. The two agree today because the badge
// happens to be nine cells wide and the number is ten. Nothing enforces that:
// a badge one word longer is still a badge, and the last cell of it would
// quietly start filtering the board to To Do instead of clearing the filters.
//
// So the property is asked of every cell the badge actually occupies.
func TestClickingAnywhereOnTheBadgeResetsTheFilters(t *testing.T) {
	m, _ := testModel(t, 150, 30)

	badge := lipgloss.Width(m.name())
	if badge == 0 {
		t.Fatal("the name badge rendered to nothing")
	}

	for x := range badge {
		got := m.hitHeader(x, m.frame.Header.Y)
		if got.Kind != TargetHeaderField || got.Field != "orbit" {
			t.Errorf("column %d of the %d-cell badge hits %+v, not the badge; hitHeader and name() disagree about how wide it is", x, badge, got)
		}
	}
}

// TestTheBadgeIsNotWrappedInBrackets. The pill paints its own background,
// which is what says where the badge begins and ends, and every other
// clickable thing on that line is a pill with no brackets inside it.
//
// The escapes come off first. Every ANSI sequence starts ESC '[', so asking
// the painted string whether it contains a bracket is a question that answers
// yes no matter what the badge says.
func TestTheBadgeIsNotWrappedInBrackets(t *testing.T) {
	m, _ := testModel(t, 150, 30)

	text := ansi.Strip(m.name())
	if strings.Contains(text, "[") || strings.Contains(text, "]") {
		t.Errorf("the name badge reads %q, which still carries the brackets the pill makes unnecessary", text)
	}

	if !strings.Contains(text, "orbit") {
		t.Errorf("the name badge reads %q and does not say orbit", text)
	}
}

// TestTheBadgeIsLitWhenNothingIsFilteredOut. Each queue badge lights up when
// its queue is the one on screen. The badge that means "all of them" did not,
// so the one arrangement of the board with nothing held back was the one with
// nothing lit — and the reader who had just clicked it got no answer back.
func TestTheBadgeIsLitWhenNothingIsFilteredOut(t *testing.T) {
	m, _ := testModel(t, 150, 30)

	clear := m.name()
	if clear != PillSelected("◉ orbit", "#FFFFFF", "#0F766E") {
		t.Error("the board is holding nothing back and the name badge is not lit")
	}

	band := view.Running
	m.queueFilter = &band

	if held := m.name(); held == clear {
		t.Error("a queue is filtering the board and the name badge is still lit as if it were not")
	}

	m.queueFilter = nil
	m.filter = "pay"

	if typed := m.name(); typed == clear {
		t.Error("a search is filtering the board and the name badge is still lit as if it were not")
	}

	m.filter = ""
	m.repoFilter = "/somewhere"

	if scoped := m.name(); scoped == clear {
		t.Error("a repository is filtering the board and the name badge is still lit as if it were not")
	}
}

// TestLightingTheBadgeDoesNotMoveIt. Being lit must not change the badge's
// width, because hitHeader places the four queue badges after it by columns
// written down in target.go: two extra cells here and every one of them is
// clicked two cells off, in the state a reader reaches by clicking this very
// badge.
func TestLightingTheBadgeDoesNotMoveIt(t *testing.T) {
	m, _ := testModel(t, 150, 30)

	lit := lipgloss.Width(m.name())

	band := view.Done
	m.queueFilter = &band

	if dim := lipgloss.Width(m.name()); dim != lit {
		t.Errorf("the name badge is %d cells lit and %d cells not; every queue badge after it moves by %d", lit, dim, lit-dim)
	}
}

// TestEachQueueBadgeCarriesItsOwnCount.
//
// board.Counts is indexed by view.Band — ToDo, NeedsYou, Running, Done — and
// the badges are drawn in a different order, ToDo, Running, NeedsYou, Done.
// Walking Counts 0..3 down the drawn list therefore hands the Running badge
// the number of tasks waiting on the reader, and Needs You the number still
// running. Both numbers are plausible, so the swap reads as true.
//
// Four different counts is what makes the swap visible: with any two of them
// equal, the arrangement that reports them backwards passes.
func TestEachQueueBadgeCarriesItsOwnCount(t *testing.T) {
	m, _ := testModel(t, 150, 30)

	want := map[view.Band]int{view.ToDo: 1, view.NeedsYou: 2, view.Running: 3, view.Done: 4}
	for band, n := range want {
		m.board.Counts[band] = n
	}

	badges := m.queueBadges()
	if len(badges) != len(want) {
		t.Fatalf("queueBadges() drew %d badges, want %d", len(badges), len(want))
	}

	for _, b := range badges {
		text := ansi.Strip(b.text)
		if !strings.HasSuffix(strings.TrimSpace(text), strconv.Itoa(want[b.band])) {
			t.Errorf("the %s badge reads %q and %s holds %d tasks",
				b.band, strings.TrimSpace(text), b.band, want[b.band])
		}
	}
}
