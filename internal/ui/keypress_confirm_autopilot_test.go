package ui

// keypress_coverage_test.go is three of keypress.go's own branches that the
// rest of the suite reaches only in passing: autopilot's refusal and its
// off-to-on turn, the confirm the CLI gesture opens rather than the cancel
// one every other test asks, and open()'s answer when nothing is selected
// at all.

import (
	"errors"
	"testing"
)

func TestAutopilotRefusesOrTurnsBothWays(t *testing.T) {
	// 1. A window with no settings file has nothing to flip.
	m, _ := testModel(t, 100, 30)
	m.opts.Settings = nil

	next, cmd := m.autopilot()
	if cmd != nil || asModel(t, next).message != "" {
		t.Error("autopilot with no settings port should do and say nothing")
	}

	// 2. A settings file that refuses the write says why, rather than
	// claiming the switch moved.
	m2, _ := testModel(t, 100, 30)
	s := m2.opts.Settings.(*settings) //nolint:errcheck
	s.fail = errors.New("settings write refused")
	next2, _ := m2.autopilot()
	wantBand(t, asModel(t, next2), "settings write refused")

	// 3. Off to on: the band says autopilot is on, and the switch is on.
	m3, _ := testModel(t, 100, 30)
	m3.opts.Settings.(*settings).autopilot = false //nolint:errcheck
	next3, _ := m3.autopilot()

	after3 := asModel(t, next3)
	if !after3.autopilotOn() {
		t.Error("autopilot from off should turn it on")
	}

	wantBand(t, after3, "autopilot is on")
}

func TestOpenWithNothingSelectedDoesNothing(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.cursor = -1

	next, cmd := m.open()
	if cmd != nil {
		t.Error("open with nothing selected produced a command")
	}

	if asModel(t, next).cursor != -1 {
		t.Error("open with nothing selected moved the cursor")
	}
}

func TestConfirmKeyAnswersThePostCliQuestion(t *testing.T) {
	// 1. Confirming with y opens the compose form, starting in the
	// checkout the CLI session ran in.
	m, _ := testModel(t, 100, 30)
	m.confirm, m.confirmID = confirmPostCliTask, "/checkouts/payments"

	next, cmd := m.confirmKey(press("y"))
	if cmd != nil {
		t.Error("confirming the post-CLI question produced a command")
	}

	after := asModel(t, next)
	if after.screen != screenCompose || after.compose.repoPath != "/checkouts/payments" {
		t.Errorf("confirmKey(y) after a CLI session = screen=%v repo=%q, want screenCompose on the payments checkout", after.screen, after.compose.repoPath)
	}

	// 2. The same question with no repository remembered leaves
	// openCompose's own choice alone — the cursor's task, not a forced one
	// — and leaves the caret where the form put it, on the flow.
	m2, _ := testModel(t, 100, 30)
	m2 = onRow(t, m2, "ACME-2701") // repo "app", distinct from ACME-2705's "payments"
	m2.confirm, m2.confirmID = confirmPostCliTask, ""
	next2, _ := m2.confirmKey(press("y"))

	after2 := asModel(t, next2)
	if after2.screen != screenCompose || after2.compose.repoPath != "/checkouts/app" {
		t.Errorf("confirmKey(y) with no repo = screen=%v repo=%q, want screenCompose starting in the cursor's own checkout, app", after2.screen, after2.compose.repoPath)
	}

	if after2.compose.field != composeFlow {
		t.Errorf("confirmKey(y) with no repo left the caret on field %v, want composeFlow", after2.compose.field)
	}

	// 3. Anything else answers no: the session is simply over.
	m3, _ := testModel(t, 100, 30)
	m3.confirm, m3.confirmID = confirmPostCliTask, "payments"
	next3, cmd3 := m3.confirmKey(press("n"))

	after3 := asModel(t, next3)
	if cmd3 != nil || after3.screen == screenCompose {
		t.Error("declining the post-CLI question should not open compose")
	}

	wantBand(t, after3, "interactive session ended")
}

func TestConfirmKeyAnswersCancelAndForgetsAGoneTask(t *testing.T) {
	// 1. A no answers nothing and asks nothing of the control port.
	m, _ := testModel(t, 100, 30)
	m.confirm, m.confirmID = confirmCancel, "ACME-2705"

	next, cmd := m.confirmKey(press("n"))
	if cmd != nil {
		t.Error("declining cancel produced a command")
	}

	if asModel(t, next).confirm != confirmNone {
		t.Error("confirmKey did not close the question either way")
	}

	// 2. A yes for a task that has since left the board does nothing —
	// there is nothing left to write "cancel" against.
	m2, _ := testModel(t, 100, 30)
	m2.confirm, m2.confirmID = confirmCancel, "ACME-GONE"

	next2, cmd2 := m2.confirmKey(press("y"))
	if cmd2 != nil {
		t.Error("confirming cancel on a task no longer on the board produced a command")
	}

	_ = next2
}
