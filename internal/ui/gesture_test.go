package ui

// gesture_test.go is the seven keys that reach the functions the subcommands
// call, and the one structural claim that holds the whole message layer
// together.
//
// Nothing here runs a process. The one gesture that would — t, which suspends
// the window and hands the terminal to the engine — is asserted on as a
// command line: the *exec.Cmd it builds is read, and never started. Running
// it would open a session and spend money, which is not a thing a test suite
// may do.

import (
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// parked is the fixture board with one run stopped at a phase boundary,
// which is the state four of these gestures are about and which no row of
// the shipped fixture is in.
func parkedModel(t *testing.T) (Model, *recorder) {
	t.Helper()
	tasks := fixtureTasks()
	for i := range tasks {
		if tasks[i].ID == "ACME-2705" {
			tasks[i].Reason = view.Reason{Key: view.ReasonHeld}
		}
	}
	got := &recorder{}
	return modelWith(t, printerFor(t, "en"), fixtureBoard(tasks, 4), 100, 30, got), got
}

// takenModel is the same board with the keyboard already taken on ACME-2705:
// t pressed, the session command built, and the window told it was issued.
func takenModel(t *testing.T) (Model, *recorder) {
	t.Helper()
	m, got := parkedModel(t)
	m = onRow(t, m, "ACME-2705")
	after, cmd := advance(t, m, press("t"))
	if cmd == nil {
		t.Fatal("t on a parked task answered with no command")
	}
	after, _ = advance(t, after, cmd())
	if !after.taken["ACME-2705"] {
		t.Fatal("the window does not remember that the keyboard was taken")
	}
	return after, got
}

func TestTheGesturesReachTheFunctionsTheSubcommandsCall(t *testing.T) {
	cases := []struct {
		name  string
		start func(t *testing.T) (Model, *recorder)
		keys  []string
		want  func(t *testing.T, m Model, cmd tea.Cmd, got *recorder)
	}{{
		name: "p writes pause",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			return onRow(t, m, "ACME-2705"), got
		},
		keys: []string{"p"},
		want: func(t *testing.T, _ Model, cmd tea.Cmd, got *recorder) {
			wantControl(t, cmd, got, "ACME-2705", "pause")
		},
	}, {
		name: "r writes resume on a task that is held",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := parkedModel(t)
			return onRow(t, m, "ACME-2705"), got
		},
		keys: []string{"r"},
		want: func(t *testing.T, _ Model, cmd tea.Cmd, got *recorder) {
			wantControl(t, cmd, got, "ACME-2705", "resume")
		},
	}, {
		name: "x asks first and then writes cancel",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			return onRow(t, m, "ACME-2705"), got
		},
		keys: []string{"x", "y"},
		want: func(t *testing.T, _ Model, cmd tea.Cmd, got *recorder) {
			wantControl(t, cmd, got, "ACME-2705", "cancel")
		},
	}, {
		name:  "h writes continue once the keyboard has been taken",
		start: takenModel,
		keys:  []string{"h"},
		want: func(t *testing.T, _ Model, cmd tea.Cmd, got *recorder) {
			wantControl(t, cmd, got, "ACME-2705", "continue")
		},
	}, {
		name: "h before t is refused, and the refusal names the key that takes it",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := parkedModel(t)
			return onRow(t, m, "ACME-2705"), got
		},
		keys: []string{"h"},
		want: func(t *testing.T, m Model, cmd tea.Cmd, got *recorder) {
			if cmd != nil || got.word != "" {
				t.Fatalf("cmd=%v word=%q, want nothing written for a session nobody took", cmd != nil, got.word)
			}
			// The whole sentence up to and including the key, because
			// this case is named for the key being named: every refusal
			// this window has contains the letter t somewhere.
			wantBand(t, m, "nobody took the keyboard here; press t")
		},
	}, {
		name: "d marks a finished task read",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			return onRow(t, m.expand(view.Done), "ACME-2690"), got
		},
		keys: []string{"d"},
		want: func(t *testing.T, m Model, cmd tea.Cmd, got *recorder) {
			if cmd == nil {
				t.Fatal("d answered with no command")
			}
			after, _ := advance(t, m, cmd())
			if got.read != "ACME-2690" {
				t.Errorf("the read port was asked for %q, want ACME-2690", got.read)
			}
			wantBand(t, after, "ACME-2690")
		},
	}, {
		name: "A flips the standing switch and says which way it went",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			return onRow(t, m, "ACME-2705"), got
		},
		keys: []string{"A"},
		want: func(t *testing.T, m Model, _ tea.Cmd, _ *recorder) {
			if m.autopilotOn() {
				t.Error("autopilot is still on, want it flipped off")
			}
			wantBand(t, m, "autopilot")
		},
	}, {
		name: "a is offered nowhere and refused with its reason",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			return onRow(t, m, "ACME-2705"), got
		},
		keys: []string{"a"},
		want: func(t *testing.T, m Model, cmd tea.Cmd, _ *recorder) {
			if cmd != nil {
				t.Fatal("a produced a command, and orbit cannot ask an engine anything yet")
			}
			wantBand(t, m, "cannot ask an engine a question yet")
		},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, got := c.start(t)
			var cmd tea.Cmd
			for _, k := range c.keys {
				m, cmd = advance(t, m, press(k))
			}
			c.want(t, m, cmd, got)
		})
	}
}

// wantControl executes one gesture's command and checks the word it wrote.
func wantControl(t *testing.T, cmd tea.Cmd, got *recorder, id, word string) {
	t.Helper()
	if cmd == nil {
		t.Fatalf("the gesture for %q answered with no command", word)
	}
	if _, ok := cmd().(controlMsg); !ok {
		t.Fatalf("the gesture for %q raised something other than a controlMsg", word)
	}
	if got.id != id || got.word != word {
		t.Errorf("the control port was asked to write %q on %q, want %q on %q", got.word, got.id, word, id)
	}
}

// TestTakingTheKeyboardIsRefusedWhileARunnerHoldsTheWorktree is the refusal
// that is about two writers in one checkout rather than about a session.
func TestTakingTheKeyboardIsRefusedWhileARunnerHoldsTheWorktree(t *testing.T) {
	m, got := testModel(t, 100, 30)
	m = onRow(t, m, "ACME-2705")
	after, cmd := advance(t, m, press("t"))
	if cmd != nil {
		t.Fatal("t on a live task produced a command, and no session may be opened in a worktree a phase is writing in")
	}
	if got.taken != "" {
		t.Errorf("the take port was asked about %q, want it never reached", got.taken)
	}
	// The reason and the key, each as a phrase of its own: "p" alone was
	// true of almost every refusal this window can produce.
	for _, want := range []string{"a phase is writing in this worktree", "press p to stop it"} {
		wantBand(t, after, want)
	}
}

// TestTakingTheKeyboardForksTheSessionAndRunsNothing reads the command line t
// would suspend the window for, and never starts it.
//
// --fork-session is the assertion that matters: without it the interactive
// session continues the runner's own transcript, and the run that is still
// using it would find its history rewritten underneath it.
func TestTakingTheKeyboardForksTheSessionAndRunsNothing(t *testing.T) {
	m, _ := parkedModel(t)
	m = onRow(t, m, "ACME-2705")
	after, cmd := advance(t, m, press("t"))
	if cmd == nil {
		t.Fatal("t on a parked task answered with no command")
	}
	msg, ok := cmd().(sessionMsg)
	if !ok {
		t.Fatalf("t raised %T, want a sessionMsg carrying the command line", cmd())
	}
	if msg.Err != nil || msg.Cmd == nil {
		t.Fatalf("err=%v cmd=%v, want a command line to look at", msg.Err, msg.Cmd != nil)
	}
	for _, want := range []string{"--resume", "--fork-session"} {
		if !slices.Contains(msg.Cmd.Args, want) {
			t.Errorf("the session command is %v, want %q in it", msg.Cmd.Args, want)
		}
	}
	if msg.Cmd.Dir == "" {
		t.Error("the session would run in whatever directory the window was started in, want the task's worktree")
	}
	// The command is handed back to the window, which answers with the
	// suspend. That command is asserted on and never executed: executing it
	// is what opens a session and spends money.
	next, exec := advance(t, after, msg)
	if exec == nil {
		t.Fatal("the window did not answer the session with a command to suspend for")
	}
	if !next.taken["ACME-2705"] {
		t.Error("the window did not remember that the keyboard was taken, so h will refuse")
	}
}
