package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// spentBoard is a board of finished tasks that together cost what is asked.
func spentBoard(each float64, n int) []view.Task {
	var tasks []view.Task

	for i := range n {
		tasks = append(tasks, view.Task{
			Repo: "payments", ID: "ACME-" + string(rune('1'+i)), Title: "done and paid for",
			Band: view.Done, Since: ago(time.Duration(n-i) * time.Minute), Cost: each,
		})
	}

	return tasks
}

// TestTheQueueStopsWhenTheWorkspaceHasSpentItsBudget. Autopilot is what
// makes a board run without anybody watching it, so the number that stops it
// has to be enforced where it picks the next task up.
func TestTheQueueStopsWhenTheWorkspaceHasSpentItsBudget(t *testing.T) {
	m, rec := testModel(t, 100, 30)
	m.seen = true

	held, ok := m.opts.Settings.(*settings)
	if !ok {
		t.Fatalf("the window's settings port is %T, want the fixture's", m.opts.Settings)
	}

	held.budget = 1.00

	tasks := append(spentBoard(0.50, 2), view.Task{
		Repo: "payments", ID: "ACME-9", Title: "waiting its turn", Band: view.ToDo, Since: ago(time.Minute),
	})

	next, cmd := m.applyBoard(boardMsg{Board: fixtureBoard(tasks, 1)})
	if cmd != nil {
		cmd()
	}

	if rec.flow != "" {
		t.Error("autopilot started a task on a board that has spent its budget")
	}

	if !strings.Contains(ansi.Strip(asModel(t, next).headerLine(200)), "budget") {
		t.Errorf("the header does not say the budget brake is on:\n%s", ansi.Strip(asModel(t, next).headerLine(200)))
	}
}

// TestTheQueueGoesOnUnderTheBudget. A brake that is on before the money is
// spent is a brake nobody can turn off.
func TestTheQueueGoesOnUnderTheBudget(t *testing.T) {
	m, rec := testModel(t, 100, 30)
	m.seen = true

	held, ok := m.opts.Settings.(*settings)
	if !ok {
		t.Fatalf("the window's settings port is %T, want the fixture's", m.opts.Settings)
	}

	held.budget = 10.00

	tasks := append(spentBoard(0.50, 2), view.Task{
		Repo: "payments", ID: "ACME-9", Title: "waiting its turn", Band: view.ToDo, Since: ago(time.Minute),
	})

	next, cmd := m.applyBoard(boardMsg{Board: fixtureBoard(tasks, 1)})
	if cmd == nil {
		t.Fatal("autopilot under its budget started nothing")
	}

	cmd()

	if rec.flow == "" {
		t.Error("autopilot under its budget did not start the waiting task")
	}

	_ = asModel(t, next)
}

// TestTheQueueStopsWhenTheQuotaWindowIsNearlySpent. Under a subscription
// there is no bill to cap, and the window is the whole of what running out
// means: the floor is the same brake as the workspace budget, in the unit
// that engine is paid in.
func TestTheQueueStopsWhenTheQuotaWindowIsNearlySpent(t *testing.T) {
	m, rec := testModel(t, 100, 30)
	m.seen = true

	held, ok := m.opts.Settings.(*settings)
	if !ok {
		t.Fatalf("the window's settings port is %T, want the fixture's", m.opts.Settings)
	}

	held.floor = 20

	m.opts.Quota = func(engine string) QuotaReading {
		return QuotaReading{
			Engine: engine, Sourced: true,
			Windows: []QuotaWindow{{Key: "5h", Label: "5h", Pct: 85, ResetsIn: time.Hour}},
		}
	}

	tasks := []view.Task{{
		Repo: "payments", ID: "ACME-9", Title: "waiting its turn", Band: view.ToDo, Since: ago(time.Minute),
	}}

	next, cmd := m.applyBoard(boardMsg{Board: fixtureBoard(tasks, 1)})
	if cmd != nil {
		cmd()
	}

	if rec.flow != "" {
		t.Error("autopilot started a task with 15% of the window left and a floor of 20")
	}

	if !strings.Contains(ansi.Strip(asModel(t, next).headerLine(200)), "quota") {
		t.Errorf("the header does not say the quota floor is holding the queue:\n%s", ansi.Strip(asModel(t, next).headerLine(200)))
	}
}

// TestAnEngineThatChargesIsNotHeldByAQuotaFloor. Money is that engine's
// unit and the workspace budget is its brake; a window it does not have
// cannot be nearly spent.
func TestAnEngineThatChargesIsNotHeldByAQuotaFloor(t *testing.T) {
	m, rec := testModel(t, 100, 30)
	m.seen = true

	held, ok := m.opts.Settings.(*settings)
	if !ok {
		t.Fatalf("the window's settings port is %T, want the fixture's", m.opts.Settings)
	}

	held.floor = 20

	m.opts.Quota = func(engine string) QuotaReading {
		return QuotaReading{Engine: engine, Money: true, Sourced: true}
	}

	tasks := []view.Task{{
		Repo: "payments", ID: "ACME-9", Title: "waiting its turn", Band: view.ToDo, Since: ago(time.Minute),
	}}

	_, cmd := m.applyBoard(boardMsg{Board: fixtureBoard(tasks, 1)})
	if cmd == nil {
		t.Fatal("autopilot started nothing for an engine that has no window to run out of")
	}

	cmd()

	if rec.flow == "" {
		t.Error("a quota floor held back an engine that is paid per token")
	}
}
