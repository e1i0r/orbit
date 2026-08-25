package ui

// band_cells_coverage_test.go is the activity band's own sentences —
// engineAndModel, controlSaid and commandSaid, none of which a keypress test
// reaches on every branch — and the row-drawing primitives in cells.go.

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

func TestEngineAndModel(t *testing.T) {
	tests := []struct {
		engine, model, want string
	}{
		{"claude", "opus", "claude/opus"},
		{"claude", "", "claude"},
		{"", "opus", "opus"},
		{"", "", ""},
	}
	for _, tt := range tests {
		got := engineAndModel(view.Task{Engine: tt.engine, Model: tt.model})
		if got != tt.want {
			t.Errorf("engineAndModel(engine=%q, model=%q) = %q, want %q", tt.engine, tt.model, got, tt.want)
		}
	}
}

func TestControlSaid(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	tests := []struct {
		word, want string
	}{
		{"pause", "asked X-1 to pause"},
		{"resume", "asked X-1 to resume"},
		{"continue", "asked X-1 to continue"},
		{"cancel", "asked X-1 to cancel"},
		{"read", "asked X-1 to read"}, // not one of the four; the fallback sentence
	}
	for _, tt := range tests {
		got := m.controlSaid(controlMsg{ID: "X-1", Word: tt.word})
		if got != tt.want {
			t.Errorf("controlSaid(%q) = %q, want %q", tt.word, got, tt.want)
		}
	}
	if got := m.controlSaid(controlMsg{ID: "X-1", Word: "pause", Err: errors.New("denied")}); got != "denied" {
		t.Errorf("controlSaid with an error = %q, want the error verbatim", got)
	}
}

func TestCommandSaid(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	if got := m.commandSaid(commandMsg{Name: "sync"}); got != "sync finished" {
		t.Errorf("commandSaid = %q, want %q", got, "sync finished")
	}
	if got := m.commandSaid(commandMsg{Name: "sync", Err: errors.New("broke")}); got != "broke" {
		t.Errorf("commandSaid with an error = %q, want the error verbatim", got)
	}
}

// TestFilterLineBranches walks filterLine through a queue filter, a typed
// filter, a repo filter, and the placeholder shown while filtering has
// begun but nothing has been typed yet.
func TestFilterLineBranches(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m.filtering = true
	if got := m.filterLine(); !strings.Contains(got, "repository, id or title") {
		t.Errorf("filterLine while filtering with nothing typed = %q, want the placeholder", got)
	}

	m.filter = "acme"
	m.filtering = false
	band := view.Running
	m.queueFilter = &band
	m.repoFilter = "payments"
	got := m.filterLine()
	for _, want := range []string{"RUNNING", "acme", "payments", "clears it"} {
		if !strings.Contains(got, want) {
			t.Errorf("filterLine = %q, want it to mention %q", got, want)
		}
	}
}

// TestBandLeftPriority walks bandLeft's order of preference: filtering,
// then the two confirmations, then a live message, then a filter already
// set, then a running task, then idle.
func TestBandLeftPriority(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m.filtering = true
	if !strings.Contains(m.bandLeft(), "repository") {
		t.Errorf("bandLeft while filtering = %q, want the filter line", m.bandLeft())
	}
	m.filtering = false

	m.confirm, m.confirmID = confirmCancel, "X-1"
	if !strings.Contains(m.bandLeft(), "X-1") {
		t.Errorf("bandLeft with confirmCancel = %q, want it to name X-1", m.bandLeft())
	}
	m.confirm = confirmPostCliTask
	if !strings.Contains(m.bandLeft(), "press y") {
		t.Errorf("bandLeft with confirmPostCliTask = %q, want the confirmation sentence", m.bandLeft())
	}
	m.confirm = confirmNone

	m.message, m.messageAt = "a fresh message", m.now
	if m.bandLeft() != Paint(Accent).Render("a fresh message") {
		t.Errorf("bandLeft with a fresh message = %q, want it painted and shown", m.bandLeft())
	}
	m.message = ""

	m.filter = "acme"
	if !strings.Contains(m.bandLeft(), "acme") {
		t.Errorf("bandLeft with a filter set = %q, want the filter line", m.bandLeft())
	}
	m.filter = ""

	// A running task in the board takes priority over the idle line.
	if !strings.Contains(m.bandLeft(), "ACME-2705") && !strings.Contains(m.bandLeft(), "ACME-2706") {
		t.Errorf("bandLeft with a task running = %q, want it to name the running task", m.bandLeft())
	}
}

func TestStateWordEveryReason(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	tests := []struct {
		task view.Task
		want string
	}{
		{view.Task{Reason: view.Reason{Key: view.ReasonFailed}}, "failed"},
		{view.Task{Reason: view.Reason{Key: view.ReasonFailedToStart}}, "would not start"},
		{view.Task{Reason: view.Reason{Key: view.ReasonGate}}, "waiting"},
		{view.Task{Reason: view.Reason{Key: view.ReasonHeld}}, "held"},
		{view.Task{Reason: view.Reason{Key: view.ReasonTimedOut}}, "timed out"},
		{view.Task{Reason: view.Reason{Key: view.ReasonAbandoned}}, "abandoned"},
		{view.Task{Reason: view.Reason{Key: view.ReasonCancelled}}, "cancelled"},
		{view.Task{Damaged: 1}, "unreadable"},
		{view.Task{Band: view.Running}, "running"},
		{view.Task{Band: view.Done}, "finished"},
		{view.Task{Band: view.ToDo}, "not started"},
	}
	for _, tt := range tests {
		word, _ := m.stateWord(tt.task)
		if !strings.Contains(word, tt.want) {
			t.Errorf("stateWord(%+v) = %q, want it to mention %q", tt.task, word, tt.want)
		}
	}
}

func TestPhaseWordWithAndWithoutFraction(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.totals = map[string]int{"careful": 3}

	if got := m.phaseWord(view.Task{Flow: "careful"}); !strings.Contains(got, "running") {
		t.Errorf("phaseWord with no phase = %q, want the plain running word", got)
	}
	if got := m.phaseWord(view.Task{Flow: "careful", Phase: "review", PhaseN: 1}); got != "review" {
		t.Errorf("phaseWord on the first phase = %q, want the bare phase name", got)
	}
	if got := m.phaseWord(view.Task{Flow: "careful", Phase: "review", PhaseN: 2}); !strings.Contains(got, "2/3") {
		t.Errorf("phaseWord past the first phase = %q, want a 2/3 fraction", got)
	}
	if got := m.phaseWord(view.Task{Flow: "unknown-flow", Phase: "review", PhaseN: 2}); got != "review" {
		t.Errorf("phaseWord with no known total = %q, want the bare phase name", got)
	}
}

func TestElapsedEveryUnit(t *testing.T) {
	now := fixtureNow
	tests := []struct {
		since time.Time
		want  string
	}{
		{time.Time{}, ""},
		{now.Add(5 * time.Second), "0s"}, // clamped: since is after now
		{now.Add(-30 * time.Second), "30s"},
		{now.Add(-5 * time.Minute), "5m"},
		{now.Add(-3 * time.Hour), "3h"},
		{now.Add(-50 * time.Hour), "2d"},
	}
	for _, tt := range tests {
		if got := elapsed(now, tt.since); got != tt.want {
			t.Errorf("elapsed(now, %v) = %q, want %q", tt.since, got, tt.want)
		}
	}
}

func TestPadEdgeCases(t *testing.T) {
	if got := pad("x", 0, false); got != "" {
		t.Errorf("pad with 0 cells = %q, want empty", got)
	}
	if got := pad("hi", 5, true); got != "   hi" {
		t.Errorf("pad right-aligned = %q, want %q", got, "   hi")
	}
	if got := pad("hi", 5, false); got != "hi   " {
		t.Errorf("pad left-aligned = %q, want %q", got, "hi   ")
	}
}

func TestBandNameEveryBand(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	for _, b := range []view.Band{view.NeedsYou, view.Running, view.ToDo, view.Done} {
		if m.bandName(b) == "" {
			t.Errorf("bandName(%v) is empty", b)
		}
	}
	if got := m.bandName(view.Band(99)); got != "" {
		t.Errorf("bandName(99) = %q, want empty for an unknown band", got)
	}
}

func TestHeadHintBranches(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. A shut band advertises the key that opens it.
	m.expanded[view.Done] = false
	if got := m.headHint(row{band: view.Done, n: 3}); got == "" {
		t.Error("headHint on a shut band is empty, want the open hint")
	}

	// 2. An open band with nothing special to say is silent.
	m.expanded[view.Done] = true
	if got := m.headHint(row{band: view.Done, n: 3}); got != "" {
		t.Errorf("headHint on an open ordinary band = %q, want empty", got)
	}

	// 3. The To Do band at the unread cap says so, ahead of the open hint.
	m.expanded[view.ToDo] = false
	m.opts.Settings = &settings{autopilot: true, lang: "en", unread: 1}
	if got := m.headHint(row{band: view.ToDo, n: 4}); !strings.Contains(got, "unread cap") {
		t.Errorf("headHint on To Do at the cap = %q, want the unread cap sentence", got)
	}
}

func TestPhaseTotalsUnknownFlow(t *testing.T) {
	totals := phaseTotals([]view.Task{
		{Flow: ""},
		{Flow: "not-a-real-flow"},
		{Flow: "task"},
		{Flow: "task"}, // repeated: the second lookup must be skipped, not repeated
	})
	if _, ok := totals["not-a-real-flow"]; ok {
		t.Error("phaseTotals resolved a flow name that does not exist")
	}
	if totals["task"] == 0 {
		t.Error("phaseTotals did not resolve the built-in task flow")
	}
}
