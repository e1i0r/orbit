package ui

// refusal_test.go is the window when something is wrong: a session that
// never opened, a task that left the board while the dialog was up, a flow
// file that will not parse, the brake with more waiting than it can name.
//
// It is a file of its own because the other tables in this package are the
// happy paths — the gesture table presses keys that work, the transition
// table walks screens that open — and these are the branches those tables
// step over. They are also the branches a reader only ever meets when
// something has already gone wrong, which is the worst moment to find out
// that a sentence is missing or that the window is remembering something
// that is not true.
//
// Nothing here starts a process. The one *exec.ExitError is built by hand
// and never came from a program: what is being asserted is which of two
// endings the window believes it is looking at, not what any engine did.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
)

// A session that never started is not a session, and the window must not go
// on believing somebody is sitting in front of the task. This is the
// promise takeKey's own doc makes, and until sessionEnded cleared the map on
// this arm it was a promise about the port's failure only.
func TestASessionThatNeverOpenedIsForgotten(t *testing.T) {
	m, _ := takenModel(t)
	after, cmd := advance(t, m, sessionEndedMsg{ID: "ACME-2705", Err: errors.New("fork/exec claude: no such file or directory")})
	if cmd != nil {
		t.Error("a session that failed to start produced a command")
	}
	if after.taken["ACME-2705"] {
		t.Error("the window still believes the keyboard was taken for a session that never opened")
	}
	wantBand(t, after, "no such file or directory")
}

// A session that ran and exited badly is the opposite case wearing the same
// word: the reader sat in front of it, and the run is still theirs to hand
// back. Telling the two apart is what the *exec.ExitError test in
// sessionEnded is for.
func TestASessionThatRanAndExitedBadlyIsStillTheReaders(t *testing.T) {
	m, _ := takenModel(t)
	after, _ := advance(t, m, sessionEndedMsg{
		ID:  "ACME-2705",
		Err: fmt.Errorf("claude exited: %w", &exec.ExitError{}),
	})
	if !after.taken["ACME-2705"] {
		t.Error("the window forgot a session the reader was actually sitting in front of")
	}
	wantBand(t, after, "claude exited")
}

// Back from a session that went fine, the band says what is still true
// rather than nothing at all: the run is stopped, and it is theirs until
// they hand it back.
func TestComingBackFromASessionSaysWhatIsStillTrue(t *testing.T) {
	m, _ := takenModel(t)
	after, _ := advance(t, m, sessionEndedMsg{ID: "ACME-2705"})
	if !after.taken["ACME-2705"] {
		t.Error("the window forgot the session as soon as the reader came back")
	}
	wantBand(t, after, "still stopped and still yours")
}

// An engine that reports no session id is a fact about that engine, not an
// error, so the port answers with no command and no error at all. The window
// says so rather than suspending itself for nothing.
func TestTakingTheKeyboardWithNoSessionToCarryOnSaysSo(t *testing.T) {
	m, _ := parkedModel(t)
	after, cmd := advance(t, m, sessionMsg{ID: "ACME-2705"})
	if cmd != nil {
		t.Error("the window suspended itself for a session that does not exist")
	}
	if after.taken["ACME-2705"] {
		t.Error("the window remembers a keyboard nobody was handed")
	}
	wantBand(t, after, "has no session to carry on")
}

// A port that refuses says why, in its own words, and nothing is remembered.
func TestTakingTheKeyboardWhenThePortRefusesSaysWhy(t *testing.T) {
	m, _ := parkedModel(t)
	after, cmd := advance(t, m, sessionMsg{ID: "ACME-2705", Err: errors.New("this task has no worktree")})
	if cmd != nil {
		t.Error("a refused take produced a command")
	}
	if after.taken["ACME-2705"] {
		t.Error("the window remembers a keyboard nobody was handed")
	}
	wantBand(t, after, "no worktree")
}

// The board moves while the dialog is up — a run started elsewhere, a task
// cancelled — and ⏎ is pressed on a task that is no longer there. Nothing is
// started, and the sentence says which task it was about.
func TestStartingATaskThatLeftTheBoardStartsNothing(t *testing.T) {
	m, got := testModel(t, 100, 30)
	m, _ = dialog(t, m, "ACME-2662")

	var left []view.Task
	for _, task := range fixtureTasks() {
		if task.ID != "ACME-2662" {
			left = append(left, task)
		}
	}
	m, _ = advance(t, m, boardMsg{Board: fixtureBoard(left, 4)})
	after, cmd := advance(t, m, press("enter"))
	if cmd != nil || got.flow != "" {
		t.Fatalf("cmd=%v flow=%q, want nothing started for a task that is gone", cmd != nil, got.flow)
	}
	wantBand(t, after, "has left the board")
}

// n with the cursor on nothing at all. An empty board is the ordinary way
// to arrive here — a state root with no tasks in it — and the sentence has
// to say what to do next rather than only that nothing happened.
func TestStartingWithNoTaskUnderTheCursorSaysWhatToDo(t *testing.T) {
	m := modelWith(t, printerFor(t, "en"), fixtureBoard(nil, 0), 100, 30, nil)
	after, cmd := advance(t, m, press("n"))
	if cmd != nil {
		t.Error("n on an empty board produced a command")
	}
	if after.screen == screenStart {
		t.Error("the dialog opened for a task that is not there")
	}
	wantBand(t, after, "orbit new")
}

// n on a run that is already going. The dialog is not opened, and the
// refusal names the key that stops it — starting a second run in one
// worktree is the thing this window exists to make hard.
func TestStartingATaskThatIsAlreadyRunningIsRefused(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onRow(t, m, "ACME-2705")
	after, cmd := advance(t, m, press("n"))
	if cmd != nil || after.screen == screenStart {
		t.Errorf("screen=%v cmd=%v, want the dialog refused", after.screen, cmd != nil)
	}
	wantBand(t, after, "is already running; press x")
}

// The brake names three and then says there are more. Three is what fits an
// activity band one row tall; the count in the same sentence is what stops a
// reader believing three is all there are.
func TestTheCapRefusalNamesThreeAndSaysHowManyMoreThereAre(t *testing.T) {
	tasks := fixtureTasks()
	for i := range tasks {
		if tasks[i].Band == view.Done {
			tasks[i].Read = false
		}
	}
	m := modelWith(t, printerFor(t, "en"), fixtureBoard(tasks, 4), 100, 30, nil)
	m.opts.Settings = &settings{autopilot: true, lang: "en", unread: 3}
	unread := board.Unread(m.board)
	if unread <= namedInRefusal {
		t.Fatalf("%d finished tasks are unread, want more than the %d the sentence names", unread, namedInRefusal)
	}

	m, _ = dialog(t, m, "ACME-2662")
	after, cmd := advance(t, m, press("enter"))
	if cmd != nil {
		t.Fatal("a run started at the cap")
	}
	// The first three by name, the ellipsis for the rest, and the count so
	// the rest is a number rather than "some".
	for _, want := range []string{"ACME-2690, ACME-2691, ACME-2692, …", "6 finished tasks"} {
		wantBand(t, after, want)
	}
	if strings.Contains(after.message, "ACME-2693") {
		t.Errorf("the refusal names a fourth task, and the band is one row: %q", after.message)
	}
}

// The refusal is said on the dialog, and the dialog does not answer d — it
// takes every keystroke while it is up. So the sentence says esc first.
// A sentence that names a key the screen it is printed on will not answer
// sends the reader to press something that does nothing.
func TestTheCapRefusalNamesAKeyTheReaderCanActuallyPressNext(t *testing.T) {
	m, got := cappedModel(t)
	m, _ = dialog(t, m, "ACME-2662")
	after, _ := advance(t, m, press("enter"))
	if after.screen != screenStart {
		t.Fatalf("screen is %v after a refusal, want the dialog still up", after.screen)
	}
	wantBand(t, after, "press esc, then d")

	// And the two keys it names are the ones that get there: esc closes the
	// dialog, and d on one of the tasks it just named marks that task read.
	// The finished band is folded when the window opens, so getting to one
	// of those ids means unfolding it first — which is ⏎ on the header, and
	// is the reason the sentence names the band's tasks rather than the
	// band.
	back, _ := advance(t, after, press("esc"))
	if back.screen != screenList {
		t.Fatalf("esc left the reader on %v, want the board", back.screen)
	}
	back, _ = advance(t, onHead(t, back, view.Done), press("enter"))
	_, cmd := advance(t, onRow(t, back, "ACME-2690"), press("d"))
	if cmd == nil {
		t.Fatal("d answered with no command on the screen the refusal sends the reader to")
	}
	if _, ok := cmd().(readMsg); !ok || got.read != "ACME-2690" {
		t.Errorf("d marked %q read, want the first task the refusal named", got.read)
	}
}

// onHead puts the cursor on one band's header row.
func onHead(t *testing.T, m Model, band view.Band) Model {
	t.Helper()
	for i, r := range m.rows() {
		if r.head && r.band == band {
			m.cursor = i
			return m
		}
	}
	t.Fatalf("no header for %v", band)
	return m
}

// A flow of the reader's own that will not parse is still a flow they have,
// and ⏎ is where they find out. The dialog carries the error rather than
// dropping it, so the refusal is the parser's own sentence and not a shrug.
func TestAFlowFileThatWillNotParseIsRefusedWhenItIsRun(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "careful.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	m, got := testModel(t, 100, 30)
	m.opts.Flows = flowDir(dir)
	m, _ = dialog(t, m, "ACME-2662") // written against careful

	if chosen := m.start.chosen(); chosen.name != "careful" || chosen.err == nil {
		t.Fatalf("the dialog opened on %q with err=%v, want careful carrying its parse error", chosen.name, chosen.err)
	}
	after, cmd := advance(t, m, press("enter"))
	if cmd != nil || got.flow != "" {
		t.Fatalf("cmd=%v flow=%q, want nothing started for a flow that will not load", cmd != nil, got.flow)
	}
	wantBand(t, after, "careful.json")
}

// The dialog drops f when there is nothing to cycle to, and the footer is
// built from the same list, so a bar that offered f would be a bar offering a
// key that moves nothing. There is no board that produces this — three flows
// ship inside the binary — so it is asserted where it is decided.
func TestOneFlowIsNotACycle(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.start = startModel{id: "ACME-2662", flows: []startFlow{{name: "task"}}}
	for _, b := range m.startBindings() {
		if b.Help().Key == m.keys.ChangeFlow.Help().Key {
			t.Error("the dialog offers f with one flow, and f would move nothing")
		}
	}
	if after := m.cycleFlow(); after.start.at != 0 {
		t.Errorf("f moved the cycle to %d, and there is one flow to be on", after.start.at)
	}
	// And a dialog with no flows at all answers with an empty flow rather
	// than reaching past the end of the list.
	m.start = startModel{id: "ACME-2662"}
	if chosen := m.start.chosen(); chosen.name != "" {
		t.Errorf("an empty cycle chose %q", chosen.name)
	}
}
