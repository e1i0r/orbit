package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

func TestDetailBandLineDeliveringAction(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	tk := view.Task{
		ID:    "ACME-101",
		Band:  view.Running,
		Since: fixtureNow.Add(-2 * time.Minute),
	}
	m.board.Tasks = append(m.board.Tasks, tk)
	m.screen, m.detail = screenDetail, tk.ID

	// 1. Delivery by supervisor.
	m.delivering = deliverPending{task: tk, verb: "CREATE PR"}

	got := ansi.Strip(m.bandLeft())
	for _, want := range []string{"ACME-101", "CREATE PR", "supervisor", "2m in"} {
		if !strings.Contains(got, want) {
			t.Errorf("bandLeft delivery supervisor = %q, want %q", got, want)
		}
	}

	// 2. Delivery by command.
	m.delivering = deliverPending{task: tk, verb: "MERGE PR", cmd: "merge"}

	got = ansi.Strip(m.bandLeft())
	for _, want := range []string{"ACME-101", "MERGE PR", "merge"} {
		if !strings.Contains(got, want) {
			t.Errorf("bandLeft delivery command = %q, want %q", got, want)
		}
	}
}

func TestDetailBandLineSupervisorThinking(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	tk := view.Task{ID: "ACME-102", Band: view.NeedsYou}
	m.board.Tasks = append(m.board.Tasks, tk)
	m.screen, m.detail = screenDetail, tk.ID

	m.supervisorBusy = true

	got := ansi.Strip(m.bandLeft())
	for _, want := range []string{"ACME-102", "supervisor is thinking..."} {
		if !strings.Contains(got, want) {
			t.Errorf("bandLeft supervisor thinking = %q, want %q", got, want)
		}
	}
}

func TestDetailBandLineRunning(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Running with active tool call.
	tkTool := view.Task{
		ID:            "ACME-103",
		Band:          view.Running,
		Live:          view.LiveHeld,
		Phase:         "implement",
		Engine:        "claude",
		Model:         "sonnet",
		Flow:          "careful",
		Since:         fixtureNow.Add(-30 * time.Second),
		CurrentAction: "git commit -m initial",
		ActionKind:    view.ActionTool,
	}
	m.board.Tasks = append(m.board.Tasks, tkTool)
	m.screen, m.detail = screenDetail, tkTool.ID

	got := ansi.Strip(m.bandLeft())
	for _, want := range []string{
		"ACME-103", "implement", "30s in", "🛠️", "git commit", "claude/sonnet", "careful",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("bandLeft running tool = %q, want %q", got, want)
		}
	}

	// 2. Running with model thought.
	tkThought := view.Task{
		ID:             "ACME-104",
		Band:           view.Running,
		Live:           view.LiveHeld,
		Phase:          "review",
		Engine:         "codex",
		Flow:           "fast",
		Since:          fixtureNow.Add(-10 * time.Second),
		CurrentThought: "analyzing test coverage\nsecond line",
	}
	m.board.Tasks = append(m.board.Tasks, tkThought)
	m.detail = tkThought.ID

	got = ansi.Strip(m.bandLeft())
	for _, want := range []string{
		"ACME-104", "review", "10s in", "🧠", "analyzing test coverage", "codex", "fast",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("bandLeft running thought = %q, want %q", got, want)
		}
	}

	// 3. Running but held/paused.
	tkHeld := view.Task{
		ID:     "ACME-105",
		Band:   view.Running,
		Live:   view.LiveHeld,
		Phase:  "implement",
		Flow:   "careful",
		Reason: view.Reason{Key: view.ReasonHeld, Args: []view.Arg{{Name: "phase", Value: "implement"}}},
	}
	m.board.Tasks = append(m.board.Tasks, tkHeld)
	m.detail = tkHeld.ID

	got = ansi.Strip(m.bandLeft())
	for _, want := range []string{"ACME-105", "held: implement", "⏸"} {
		if !strings.Contains(got, want) {
			t.Errorf("bandLeft running held = %q, want %q", got, want)
		}
	}
}

func TestDetailBandLineNeedsYou(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Gate with autopilot off.
	m.opts.Settings.(*settings).autopilot = false //nolint:errcheck
	tkGate := view.Task{
		ID:     "ACME-106",
		Band:   view.NeedsYou,
		Reason: view.Reason{Key: view.ReasonGate, Args: []view.Arg{{Name: "phase", Value: "review"}}},
		Flow:   "careful",
	}
	m.board.Tasks = append(m.board.Tasks, tkGate)
	m.screen, m.detail = screenDetail, tkGate.ID

	got := ansi.Strip(m.bandLeft())
	for _, want := range []string{"ACME-106", "waiting: review", "already waiting for you"} {
		if !strings.Contains(got, want) {
			t.Errorf("bandLeft gate without autopilot = %q, want %q", got, want)
		}
	}

	// 2. Gate with autopilot on.
	m.opts.Settings.(*settings).autopilot = true //nolint:errcheck

	got = ansi.Strip(m.bandLeft())
	for _, want := range []string{"ACME-106", "waiting: review", "lifting this gate"} {
		if !strings.Contains(got, want) {
			t.Errorf("bandLeft gate with autopilot = %q, want %q", got, want)
		}
	}

	// 3. Failed task.
	tkFailed := view.Task{
		ID:     "ACME-107",
		Band:   view.NeedsYou,
		Reason: view.Reason{Key: view.ReasonFailed, Args: []view.Arg{{Name: "phase", Value: "build"}}},
	}
	m.board.Tasks = append(m.board.Tasks, tkFailed)
	m.detail = tkFailed.ID

	got = ansi.Strip(m.bandLeft())
	if !strings.Contains(got, "failed: build") {
		t.Errorf("bandLeft failed = %q, want failed: build", got)
	}
}

func TestDetailBandLineToDoAndDone(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. To Do task.
	tkTodo := view.Task{ID: "ACME-108", Band: view.ToDo, Flow: "careful", Repo: "orbit"}
	m.board.Tasks = append(m.board.Tasks, tkTodo)
	m.screen, m.detail = screenDetail, tkTodo.ID

	got := ansi.Strip(m.bandLeft())
	for _, want := range []string{"ACME-108", "not started", "careful", "orbit"} {
		if !strings.Contains(got, want) {
			t.Errorf("bandLeft todo = %q, want %q", got, want)
		}
	}

	// 2. Done task.
	tkDone := view.Task{
		ID:     "ACME-109",
		Band:   view.Done,
		Flow:   "quick",
		Engine: "claude",
		Since:  fixtureNow.Add(-10 * time.Minute),
	}
	m.board.Tasks = append(m.board.Tasks, tkDone)
	m.detail = tkDone.ID

	got = ansi.Strip(m.bandLeft())
	for _, want := range []string{"ACME-109", "finished", "10m in", "claude", "quick"} {
		if !strings.Contains(got, want) {
			t.Errorf("bandLeft done = %q, want %q", got, want)
		}
	}
}

func TestDetailBandScreenVsListScreen(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// ACME-200 is running on the board.
	tkRunning := view.Task{
		ID:    "ACME-200",
		Band:  view.Running,
		Phase: "implement",
		Live:  view.LiveHeld,
	}
	// ACME-201 is in ToDo.
	tkTodo := view.Task{
		ID:   "ACME-201",
		Band: view.ToDo,
		Flow: "careful",
	}
	m.board.Tasks = []view.Task{tkRunning, tkTodo}

	// In screenList: board has a running task, so band shows the running task ACME-200.
	m.screen = screenList

	gotList := ansi.Strip(m.bandLeft())
	if !strings.Contains(gotList, "ACME-200") {
		t.Errorf("screenList band = %q, want it to show running task ACME-200", gotList)
	}

	// In screenDetail on ACME-201 (ToDo): band MUST show ACME-201, NOT ACME-200!
	m.screen, m.detail = screenDetail, tkTodo.ID

	gotDetail := ansi.Strip(m.bandLeft())
	if !strings.Contains(gotDetail, "ACME-201") {
		t.Errorf("screenDetail band = %q, want it to show viewed task ACME-201", gotDetail)
	}

	if strings.Contains(gotDetail, "ACME-200") {
		t.Errorf("screenDetail band = %q, must NOT show board's other task ACME-200", gotDetail)
	}
}

func TestMovingWithDeliveryAndWatch(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = nil

	if m.moving() {
		t.Error("expected moving() to be false initially")
	}

	m.delivering = deliverPending{verb: "FIX CHECKS"}
	if !m.moving() {
		t.Error("expected moving() to be true when delivering.verb is set")
	}

	m.delivering = deliverPending{}

	m.watching = &commandWatch{name: "sync"}
	if !m.moving() {
		t.Error("expected moving() to be true when watching is set")
	}
}
