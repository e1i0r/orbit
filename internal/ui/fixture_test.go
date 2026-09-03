package ui

// fixture_test.go holds the board every test in this package draws, and the
// two ports the window is handed instead of a state root: a settings file in
// memory and a control function that records rather than acts.
//
// It is a file of its own because the transition table, the measured render
// and the golden files all need the same board, and because the 300-line
// ceiling is a real ceiling — a fixture inlined into the table would have
// pushed that file over it, and a table nobody can read at one sitting is
// the thing the ceiling exists to prevent.
//
// Nothing here opens a terminal, spawns a process, or reads anything outside
// this repository.

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// fixtureNow is the clock every fixture is measured against, so an elapsed
// column is a fact about the fixture and not about when the test ran.
var fixtureNow = time.Date(2026, 8, 23, 15, 4, 0, 0, time.UTC)

func ago(d time.Duration) time.Time { return fixtureNow.Add(-d) }

// settings is the settings file, in memory. It is the whole of what
// internal/cli will satisfy with a store-backed type.
type settings struct {
	autopilot bool
	lang      string
	unread    int
	budget    float64
	floor     int
	fail      error
}

func (s *settings) Autopilot() bool { return s.autopilot }

func (s *settings) SetAutopilot(v bool) error {
	if s.fail != nil {
		return s.fail
	}

	s.autopilot = v

	return nil
}

func (s *settings) Language() string { return s.lang }

func (s *settings) SetLanguage(v string) error {
	if s.fail != nil {
		return s.fail
	}

	s.lang = v

	return nil
}

// The rest keep no value — nothing reads one back — but every one of them
// answers fail, because every one of them can refuse in the file this
// stands for: the settings file has a lock, and a second orbit holding it
// makes any of these say so after waiting two seconds.
func (s *settings) UnreadCap() int           { return s.unread }
func (s *settings) BudgetWorkspace() float64 { return s.budget }
func (s *settings) QuotaFloor() int          { return s.floor }
func (s *settings) SetUnreadCap(int) error   { return s.fail }
func (s *settings) Engine() string           { return "" }
func (s *settings) SetEngine(string) error   { return s.fail }
func (s *settings) Model() string            { return "" }
func (s *settings) SetModel(string) error    { return s.fail }
func (s *settings) Flow() string             { return "task" }
func (s *settings) SetFlow(string) error     { return s.fail }
func (s *settings) Theme() string            { return "monokai" }
func (s *settings) SetTheme(string) error    { return s.fail }

// arg is one placeholder for a reason, spelled the way internal/view spells
// it so a fixture reads like the record it stands for.
func arg(name, value string) view.Arg { return view.Arg{Name: name, Value: value} }

// fixtureTasks is the board the plan's 100-column screen draws, in the
// order that screen draws it.
func fixtureTasks() []view.Task {
	tasks := []view.Task{
		{
			Repo: "payments", ID: "ACME-2662", Title: "Retry the webhook on 5xx", Band: view.NeedsYou,
			Flow: "careful", Phase: "gates", Attempt: 2, Since: ago(31 * time.Minute),
			Reason: view.Reason{Key: view.ReasonFailed, Args: []view.Arg{arg("phase", "gates")}},
		},
		{
			Repo: "app", ID: "ACME-2701", Title: "Move the assets cron", Band: view.NeedsYou,
			Flow: "task", Phase: "review", Attempt: 1, Since: ago(4 * time.Minute),
			Reason: view.Reason{Key: view.ReasonGate, Args: []view.Arg{arg("phase", "review")}},
		},
		{
			Repo: "payments", ID: "ACME-2698", Title: "Fix the swagger lint", Band: view.NeedsYou,
			Flow: "task", Attempt: 1, Since: ago(3 * time.Hour),
			Reason: view.Reason{Key: view.ReasonAbandoned},
		},
		// The one task on this board that reaches past the repository it
		// was written in. Its row is a row like any other and its column
		// says so: `app +2`, three checkouts, one task.
		{
			Repo: "app", Repos: []string{"app", "payments", "api"},
			ID: "ACME-2705", Title: "Reconciliation endpoint", Band: view.Running,
			Flow: "careful", Phase: "implement", PhaseN: 1, Engine: "claude", Model: "opus",
			Live: view.LiveHeld, Attempt: 1, Since: ago(8 * time.Minute), Started: ago(8 * time.Minute),
		},
		{
			Repo: "payments", ID: "ACME-2706", Title: "Index on settlements", Band: view.Running,
			Flow: "careful", Phase: "review", PhaseN: 2, Engine: "claude", Model: "opus",
			Live: view.LiveHeld, Attempt: 1, Since: ago(3 * time.Minute), Started: ago(40 * time.Minute),
		},
	}
	for _, id := range []string{"ACME-2710", "ACME-2711", "ACME-2712", "ACME-2713"} {
		tasks = append(tasks, view.Task{
			Repo: "app", ID: id, Title: "Written down and not started",
			Band: view.ToDo, Flow: "task", Since: ago(2 * time.Hour),
		})
	}

	for i, id := range []string{"ACME-2690", "ACME-2691", "ACME-2692", "ACME-2693", "ACME-2694", "ACME-2695"} {
		tasks = append(tasks, view.Task{
			Repo: "payments", ID: id, Title: "Finished earlier today",
			Band: view.Done, Flow: "task", Phase: "review", Engine: "claude", Model: "opus",
			Attempt: 1, Read: i >= 3, Since: ago(time.Duration(i+1) * time.Hour),
		})
	}

	// Every row knows where its home repository is, as every row a Reader
	// builds does. The repositories a task was carried into are named and
	// not located, which is also true of the rows: view.Task.Repos is a
	// list of names, and the one path on it is the home one's.
	for i := range tasks {
		tasks[i].RepoPath = "/checkouts/" + tasks[i].Repo
	}

	return tasks
}

// fixtureBoard is what a Reader would have handed the window, counts and
// all. counts is computed with view.BandOf for the same reason internal/board
// computes it that way: the number above a band and the rows inside it have
// to be one answer.
func fixtureBoard(tasks []view.Task, repos int) board.Board {
	b := board.Board{Tasks: tasks, Repos: repos, ReadAt: fixtureNow}
	for _, t := range tasks {
		b.Counts[view.BandOf(t)]++
	}

	return b
}

// testModel is a window of the given size with the fixture board already in
// it, its clock stopped, and a control port that records rather than acts.
func testModel(t *testing.T, w, h int) (Model, *recorder) {
	t.Helper()

	got := &recorder{}
	m := modelWith(t, words.For("en"), fixtureBoard(fixtureTasks(), 4), w, h, got)

	return m, got
}

// lastRow is the fixture window with the cursor on the final row of the
// body, which is the precondition two rows of the transition table share:
// the one that walks off the bottom and the one whose list shrinks under it.
func lastRow(t *testing.T) Model {
	t.Helper()
	m, _ := testModel(t, 100, 30)
	m.cursor = len(m.rows()) - 1

	return m
}

// openOn is the same window with the task view already open on one id.
//
// A diff is only taken for the task the pane is open on — a late answer for
// a task the reader has left would otherwise land under another task's
// heading — so a test that hands Update a diffMsg has to be in this state
// before the message means anything at all.
func openOn(t *testing.T, id string) Model {
	t.Helper()
	m, _ := testModel(t, 100, 30)
	m.screen, m.detail = screenDetail, id

	return m
}

// modelWith is the same window in whatever language and over whatever board
// a caller names.
//
// The ports come from the recorder and the standing state is filled here, in
// one place, so that a test which is about a keystroke does not also have to
// decide what a settings file says. Flows is the one field left at its zero
// value, and deliberately: nil is the built-ins and nothing else, which is
// what a window opened without a state root has, and a test that is about a
// reader's own flows sets it itself.
func modelWith(t *testing.T, p *words.Printer, b board.Board, w, h int, got *recorder) Model {
	t.Helper()

	o := got.ports()
	o.Root, o.Settings, o.Words = "~/work", &settings{autopilot: true, lang: "en", unread: 5}, p
	o.Width, o.Height = w, h
	// Every engine in the fixture can resume. The port is a function of the
	// engine's name, and the one test that is about an engine that cannot
	// replaces it with a closure that answers for a name.
	o.CanResume = func(string) bool { return true }
	m := New(o)
	m.now = fixtureNow
	next, _ := m.Update(boardMsg{Board: b})

	loaded, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", next)
	}

	loaded.now = fixtureNow

	return loaded
}

// press is one keystroke as the event loop delivers it.
func press(keystroke string) tea.KeyPressMsg {
	switch keystroke {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	}

	r := []rune(keystroke)[0]

	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// at moves the cursor to the first row of the given band, or to its head.
//
// The blank check is not belt and braces: view.ToDo is zero, so a separator
// row's zero-value band is ToDo, and without it this would happily park the
// cursor on a blank line and every assertion after it would be about nothing.
func at(t *testing.T, m Model, b view.Band, head bool) Model {
	t.Helper()

	for i, r := range m.rows() {
		if r.band == b && r.head == head && !r.blank {
			m.cursor = i
			return m
		}
	}

	t.Fatalf("no %v row in the body", b)

	return m
}

// wantPane fails unless the task view's diff contains want, which is how a
// message that lands in the pane rather than in the band is checked.
func wantPane(t *testing.T, m Model, want string) {
	t.Helper()

	if !strings.Contains(m.diff, want) {
		t.Errorf("the diff pane says %q, want it to mention %q", m.diff, want)
	}
}

// wantNoPane fails unless the task view's diff is empty, which is what a
// diff the window was right to drop leaves behind.
func wantNoPane(t *testing.T, m Model) {
	t.Helper()

	if m.diff != "" {
		t.Errorf("the diff pane says %q, want it left alone", m.diff)
	}
}

// discriminatingWant is how short an argument to wantBand may be before the
// check stops being a check.
//
// Four runes, because the shortest want in this package that is a word is
// "diff", and because the arguments this constant exists to refuse were "t"
// and "p": single keystrokes. Every refusal in why.go contains the letter t
// somewhere, so a test named for naming the right key was green while the
// reader was being told something else entirely. A substring check is only
// worth what its substring discriminates.
const discriminatingWant = 4

// wantBand fails unless the activity band is saying something that contains
// want. The band is where a message lands, and a message that landed
// nowhere is a Cmd whose answer the reader never sees.
//
// want has to be long enough to name one message rather than any message.
// The helper's contract — "the band said something about this" — is what
// invites a one-letter argument, so the helper refuses one rather than
// leaving the next reader to notice.
func wantBand(t *testing.T, m Model, want string) {
	t.Helper()

	if utf8.RuneCountInString(want) < discriminatingWant {
		t.Fatalf("wantBand(%q) would pass on almost any sentence; assert a phrase only the right message contains", want)
	}

	if !strings.Contains(m.message, want) {
		t.Errorf("the band says %q, want it to mention %q", m.message, want)
	}
}
