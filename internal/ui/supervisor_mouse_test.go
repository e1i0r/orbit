package ui

// The supervisor screen, pointed at through the window.
//
// The screen answers for its own cells; what this file is about is the wire
// between them. The window used to answer no target at all for this screen —
// `case screenHelp, screenSupervisor, screenQuota: return point.Target{}` —
// so a screen that knows perfectly well what is under the pointer would
// still never be asked.

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// TestTheWindowAsksTheSupervisorScreenWhatIsUnderThePointer. Not what the
// answer is — that is the screen's own test — but that the question reaches
// it at all, which for three years of this file it did not.
func TestTheWindowAsksTheSupervisorScreenWhatIsUnderThePointer(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenSupervisor

	var asked bool

	for y := range 30 {
		if m.frame.At(y) != layout.RegionBody {
			continue
		}

		want := m.supervisor.Hit(4, y, m.supervisorEnv())

		if got := m.hit(4, y); got != want {
			t.Fatalf("the window answers %+v at row %d and the screen answers %+v", got, y, want)
		}

		asked = true
	}

	if !asked {
		t.Fatal("a window of thirty rows has no body to point at")
	}
}

// TestAClickOnTheSupervisorsListsReachesTheScreen. Each of the three kinds
// goes to one place, because the screen tells them apart and this window has
// nothing to add — a switch here would be its decisions written down a
// second time, one package away from the first.
func TestAClickOnTheSupervisorsListsReachesTheScreen(t *testing.T) {
	kinds := []point.Kind{
		point.SupervisorConversation,
		point.SupervisorOffer,
		point.SupervisorLine,
	}

	for _, kind := range kinds {
		m, _ := testModel(t, 100, 30)
		m.screen = screenSupervisor

		next, _ := m.leftClick(point.Target{Kind: kind, Pane: 0})

		after := asModel(t, next)
		if after.screen != screenSupervisor {
			t.Errorf("a click of kind %v left the supervisor screen", kind)
		}
	}
}

// TestTheTwoScreensWithNothingInTheirBodyStillAnswerNothing. The cheat sheet
// is a list of what other screens' keys do and the quota is a reading:
// neither has a gesture in its body, and answering nothing is what keeps a
// click from falling through to the board's rows underneath — where it
// landed on whatever task happened to be at that height, and a second one
// opened it.
func TestTheTwoScreensWithNothingInTheirBodyStillAnswerNothing(t *testing.T) {
	for _, on := range []screen{screenHelp, screenQuota} {
		m, _ := testModel(t, 100, 30)
		m.screen = on
		m.cursor = 1

		for y := range 30 {
			if m.frame.At(y) != layout.RegionBody {
				continue
			}

			if got := m.hit(4, y); got.Kind != point.None {
				t.Errorf("screen %v answers %v at row %d, want nothing", on, got.Kind, y)
			}
		}

		// And the click that does reach them is the bar's, which is
		// answered above the body's switch: the way out of every screen is
		// clickable on every screen.
		next, _ := m.leftClick(point.Target{Kind: point.BarHint, Key: "esc"})
		if asModel(t, next).cursor != m.cursor {
			t.Errorf("a bar hint on screen %v moved the board underneath", on)
		}
	}
}

// TestTheWheelAndThePointerAgreeAboutWhoOwnsTheScreen. Both were wired at
// different times and against different switches; a screen the wheel turns
// and the pointer cannot reach is half a mouse.
func TestTheWheelAndThePointerAgreeAboutWhoOwnsTheScreen(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenSupervisor
	m.cursor = 1

	turned := m.wheel(tea.MouseWheelMsg{X: 4, Y: 10, Button: tea.MouseWheelDown}.Mouse())

	if turned.cursor != m.cursor {
		t.Errorf("a notch over the supervisor moved the board to %d", turned.cursor)
	}

	if turned.screen != screenSupervisor {
		t.Error("a notch over the supervisor left the screen")
	}
}

// TestNoScreensBodyFallsThroughToTheBoardsRows.
//
// The failure this holds, written in target.go's own comment: a screen with
// no arm in the body's switch reaches hitRow underneath, and a click lands
// on whatever task happened to be at that height — a second one opens it.
// The reader is looking at the settings and ends up in a task.
//
// It walks the enum rather than a list written here, so a screen added
// without an arm is a screen this catches on the day it arrives.
func TestNoScreensBodyFallsThroughToTheBoardsRows(t *testing.T) {
	// The board is the one screen whose body is the board's rows.
	for on := screenList + 1; on <= screenKnowledge; on++ {
		m, _ := testModel(t, 100, 30)
		m.screen = on

		for y := range 30 {
			if m.frame.At(y) != layout.RegionBody {
				continue
			}

			switch got := m.hit(4, y); got.Kind {
			case point.Task, point.BandHeader:
				t.Errorf("screen %d answers %v at row %d — that is the board underneath it",
					on, got.Kind, y)
			}
		}
	}
}
