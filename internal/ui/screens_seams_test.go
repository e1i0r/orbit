package ui

// The meeting files, where the window and each screen translate for one
// another. What is checked here is the translation and nothing else: every
// screen is opened, drawn, driven by one key and closed, so a seam that
// stopped carrying something is a failure here rather than a blank screen.

import (
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/compose"
)

// TestEveryScreenOpensDrawsAndCloses. A screen the window can open and not
// draw is a blank body, and one it can draw and not close is a reader stuck
// on it.
func TestEveryScreenOpensDrawsAndCloses(t *testing.T) {
	for _, c := range []struct {
		what string
		open func(Model) Model
		want screen
	}{
		{"the quota screen", Model.openQuota, screenQuota},
		{"what Orbit knows", Model.openKnowledge, screenKnowledge},
		{"the engine knobs", Model.openEngines, screenEngines},
		{"the repository list", Model.openRepos, screenRepos},
		{"the cheat sheet", Model.openHelp, screenHelp},
		{"the supervisor", Model.openSupervisor, screenSupervisor},
		{"the form", Model.openCompose, screenCompose},
	} {
		m, _ := testModel(t, 120, 30)
		m.opts.Engines = enginesTestList
		m.opts.Quota = quotaFixture

		m = c.open(m)
		if m.screen != c.want {
			t.Errorf("%s opened on %v, want %v", c.what, m.screen, c.want)
		}

		drawn := ansi.Strip(m.View().Content)
		if strings.TrimSpace(drawn) == "" {
			t.Errorf("%s drew nothing", c.what)
		}

		left, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		if got := asModel(t, left); got.screen == c.want {
			t.Errorf("%s did not close on escape", c.what)
		}
	}
}

// TestTheBodyOfEachScreenIsDrawnByItsOwnSeam. The window asks the screen for
// its rows; a seam that lost the Env would answer with an empty one.
func TestTheBodyOfEachScreenIsDrawnByItsOwnSeam(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.opts.Engines = enginesTestList
	m.opts.Quota = quotaFixture

	for _, c := range []struct {
		what string
		rows func(Model, int, int) []string
		want string
	}{
		{"the quota screen", Model.quotaRows, "CLAUDE"},
		{"what Orbit knows", Model.knowledgeRows, "Orbit knows"},
		{"the engine knobs", Model.enginesRows, "claude"},
		{"the repository list", Model.repolistRows, "Repositories"},
		{"the cheat sheet", Model.helpRows, "BOARD"},
		{"the supervisor", Model.supervisorRows, "Supervisor"},
		{"the form", Model.composeRows, "flow:"},
	} {
		drawn := ansi.Strip(strings.Join(c.rows(m, 24, 110), "\n"))
		if !strings.Contains(drawn, c.want) {
			t.Errorf("%s does not say %q:\n%s", c.what, c.want, drawn)
		}
	}
}

// TestTheEnginesTheRosterHasAreTheOnesALineIsCredited to. A thread is a
// record and can hold an engine this build no longer has, but the roster is
// what the window knows now.
func TestTheEnginesTheRosterHasAreTheOnesALineIsCredited(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = enginesTestList

	if !m.onTheRoster("claude") {
		t.Error("an engine the port answered with is not on the roster")
	}

	if m.onTheRoster("nobody") {
		t.Error("an engine nothing answered with is on the roster")
	}
}

// TestTheWheelReachesTheSupervisorsThreadAndTheKnobsList.
func TestTheWheelReachesTheSupervisorsThreadAndTheKnobsList(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = enginesTestList
	m = m.openSupervisor()

	// Nothing said yet: scrolling is a no-op rather than a panic.
	if got := m.scrollThread(3); got.screen != screenSupervisor {
		t.Errorf("scrolling an empty thread left the window on %v", got.screen)
	}

	knobs, _ := testModel(t, 100, 30)
	knobs.opts.Engines = enginesTestList
	knobs = knobs.openEngines()

	before := ansi.Strip(strings.Join(knobs.enginesRows(20, 100), "\n"))

	after := knobs.pickEngineRow(2)
	if got := ansi.Strip(strings.Join(after.enginesRows(20, 100), "\n")); got == before {
		t.Error("the wheel over the knobs moved nothing")
	}
}

// TestATaskTheFormWroteIsRunAndThenWaitedFor. The board polls twice a
// second, so the row is selected the moment it arrives.
func TestATaskTheFormWroteIsRunAndThenWaitedFor(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	var ran []string

	m.opts.Do = func(_ string, args []string, _ io.Writer) error {
		ran = args

		return nil
	}

	next, cmd := m.writeTask(compose.Task{
		ID: "ACME-9", Repo: "/checkouts/app", Flow: "careful", Text: "write the importer", Start: true,
	})
	if cmd == nil {
		t.Fatal("writing a task ran nothing")
	}

	after := asModel(t, next)
	if after.pendingID != "ACME-9" {
		t.Errorf("the window is waiting for %q", after.pendingID)
	}

	for _, one := range commandsIn(t, cmd) {
		one()

		if ran != nil {
			break
		}
	}

	joined := strings.Join(ran, " ")
	for _, want := range []string{"-id ACME-9", "-repo /checkouts/app", "-flow careful", "-start", "write the importer"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the command line is %q, want it to carry %q", joined, want)
		}
	}

	// The row arriving is what stops the wait, and the cursor goes to it:
	// the reader wrote this task and is looking for it.
	arrived := after
	arrived.pendingID = "ACME-2706"

	landed := arrived.selectPending()
	if landed.pendingID != "" {
		t.Errorf("the window is still waiting for %q after the row arrived", landed.pendingID)
	}

	if got, ok := landed.selected(); !ok || got.task.ID != "ACME-2706" {
		t.Errorf("the cursor is on %+v, want the row that arrived", got.task.ID)
	}

	// And two refreshes with nothing arriving give up quietly: a write that
	// answered no error has nothing to apologise for.
	gave := after
	for range 3 {
		gave = gave.selectPending()
	}

	if gave.pendingID != "" {
		t.Errorf("the window is still waiting for %q after three refreshes", gave.pendingID)
	}
}

// TestAutopilotLooksAtWhatNeedsSomebodyOnceEach. The supervisor is asked
// about the tasks in needs you, and never asked about the same one twice
// while its answer is still out: two questions about one task is two engine
// calls for one answer.
func TestAutopilotLooksAtWhatNeedsSomebodyOnceEach(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Settings = &settingsFile{autopilot: true, lang: "en", unread: 9}

	var asked [][]string

	m.opts.AutoSupervise = func(_ string, ids []string) (string, error) {
		asked = append(asked, ids)

		return "looked at them", nil
	}

	next, cmd := m.autoSuperviseNeedsYou()
	if cmd == nil {
		t.Fatal("autopilot asked about nothing on a board with tasks in needs you")
	}

	if !next.supervisorBusy {
		t.Error("the window does not say it is waiting on the supervisor")
	}

	if !strings.Contains(next.message, "inspecting") {
		t.Errorf("the band says %q", next.message)
	}

	cmd()

	if len(asked) != 1 || len(asked[0]) == 0 {
		t.Fatalf("the supervisor was asked about %v", asked)
	}

	// Asked once: the same board again, with the answer still out, asks
	// nothing.
	if _, again := next.autoSuperviseNeedsYou(); again != nil {
		t.Error("autopilot asked a second time while the first answer was still out")
	}

	// And with the answer back, the tasks it already looked at are not
	// asked about again.
	done := next
	done.supervisorBusy = false

	if _, more := done.autoSuperviseNeedsYou(); more != nil {
		t.Error("autopilot asked about tasks it had already looked at")
	}
}

// TestAutopilotOffAsksNothing, and neither does a build with no supervisor
// to ask.
func TestAutopilotOffAsksNothing(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Settings = &settingsFile{lang: "en"}
	m.opts.AutoSupervise = func(string, []string) (string, error) { return "", nil }

	if _, cmd := m.autoSuperviseNeedsYou(); cmd != nil {
		t.Error("autopilot asked with the switch off")
	}

	m.opts.Settings = &settingsFile{autopilot: true, lang: "en", unread: 9}
	m.opts.AutoSupervise = nil

	if _, cmd := m.autoSuperviseNeedsYou(); cmd != nil {
		t.Error("a build with no supervisor asked one anyway")
	}
}
