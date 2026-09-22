package ui

// Where saving the compose form puts the reader.
//
// On the board, with the new row under the cursor. Not in a pane, not on
// the form, and not on a screen that has to be dismissed before the thing
// they asked for can be seen. Elio asked for this twice, which is twice
// more than a window should need telling.

import (
	"errors"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/compose"
)

// saved is the window just after the form submitted one task.
func saved(t *testing.T) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)
	m = m.openCompose()

	next, _ := m.tookCompose(m.compose, compose.Out{
		Leave: true,
		Write: &compose.Task{ID: "TST-1", Repo: "payments", Flow: "task", Text: "do a thing"},
	})

	return asModel(t, next)
}

// TestSavingTheFormLandsOnTheBoard.
func TestSavingTheFormLandsOnTheBoard(t *testing.T) {
	m := saved(t)

	if m.screen != screenList {
		t.Errorf("screen after saving = %v, want screenList", m.screen)
	}

	if m.watchUp {
		t.Error("saving the form raised the output pane; the reader asked for the board")
	}

	// The run is still going on behind it — quiet is not "not run".
	if m.watching == nil {
		t.Fatal("saving the form ran no command at all")
	}

	if m.watching.name != "board" {
		t.Errorf("the command run was %q, want board (with new as its first argument)", m.watching.name)
	}

	// And the board is told which row to move to when it arrives.
	if m.pendingID != "TST-1" {
		t.Errorf("pendingID = %q, want the id just written", m.pendingID)
	}
}

// TestTheLineTheWriteAnswersGoesOnTheBand.
//
// Quiet is about the pane, not about the answer. `board new` prints one
// line saying what it wrote, and losing it would trade a pane nobody wanted
// for a save with no confirmation at all.
func TestTheLineTheWriteAnswersGoesOnTheBand(t *testing.T) {
	m := saved(t)

	next, _ := m.Update(commandMsg{Name: "board", Text: "TST-1 written down against payments to walk task"})
	after := asModel(t, next)

	if after.watchUp {
		t.Error("a write that printed its one line still raised the pane")
	}

	if !strings.Contains(after.message, "TST-1") {
		t.Errorf("the band says %q, want the line board new answered with", after.message)
	}
}

// TestAWriteThatFailsGoesBackToTheForm is the other half, and the one that
// matters more.
//
// Success needs no reading, so it gets no screen. A failure does, and the
// screen it gets is the form: the fields hold the task, and the fields are
// what has to change. A pane here would be two gestures — dismiss it, open
// the form again — to reach the screen the reader never wanted to leave.
func TestAWriteThatFailsGoesBackToTheForm(t *testing.T) {
	m := saved(t)

	next, _ := m.Update(commandMsg{Name: "board", Err: errors.New("no such repository: payments")})
	after := asModel(t, next)

	if after.screen != screenCompose {
		t.Fatalf("screen after a failed write = %v, want the form back", after.screen)
	}

	if after.watchUp {
		t.Error("a failed write raised the output pane; the reason belongs on the form")
	}

	// Drawn on the form, where the reader is now looking.
	drawn := strings.Join(after.composeRows(30, 100), "\n")
	if !strings.Contains(drawn, "no such repository") {
		t.Errorf("the form does not say why the save failed:\n%s", drawn)
	}

	// And the band as well, for the reader who was watching that.
	if !strings.Contains(after.message, "no such repository") {
		t.Errorf("the band reads %q, want the error on it too", after.message)
	}

	// Nothing is left half-written: no pending row to chase on a board
	// that never got one.
	if after.pendingID != "" || after.writing {
		t.Errorf("after a failed write pendingID=%q writing=%v, want both cleared",
			after.pendingID, after.writing)
	}
}

// TestAFailedSaveKeepsTheFormItWasPressedOn.
//
// The form used to empty itself the moment Save was pressed, before
// anybody knew whether the write would work. A refused save then cost the
// reader the task as well as the save, which is the worst moment there is
// to lose a paragraph somebody has just written.
//
// The assertion is that the form comes back whole rather than as a blank
// one: everything it drew before Save it draws again, with the refusal
// added. A zero State draws neither the repository it was opened on nor
// the flows it was handed, so a wiped form cannot pass this.
func TestAFailedSaveKeepsTheFormItWasPressedOn(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openComposeFor("payments")

	before := m.composeRows(30, 100)

	next, _ := m.tookCompose(m.compose, compose.Out{
		Leave: true,
		Write: &compose.Task{ID: "FRA-9", Repo: "payments", Flow: "task", Text: "do a thing"},
	})

	failed, _ := asModel(t, next).Update(commandMsg{Name: "board", Err: errors.New("nope")})
	after := strings.Join(asModel(t, failed).composeRows(30, 100), "\n")

	for _, row := range before {
		if strings.TrimSpace(row) == "" {
			continue
		}

		if !strings.Contains(after, strings.TrimRight(row, " ")) {
			t.Fatalf("the form came back missing a row it had before the save:\n  gone: %q\n%s",
				row, after)
		}
	}

	if !strings.Contains(after, "nope") {
		t.Errorf("the form came back without the reason the save failed:\n%s", after)
	}
}
