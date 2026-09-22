package panes

// What the duration card counts from.

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// TestTheDurationCardMeasuresTheRunAndNotTheWait.
//
// It measured Since, which is when the state on screen began — on a task
// parked at a gate, how long it has been waiting on the reader. That read
// 1m beside a cost of $4.03 for the whole run: four dollars in one minute.
// The band already says how long the wait has been; these two cells sit on
// one row and now measure one span.
func TestTheDurationCardMeasuresTheRunAndNotTheWait(t *testing.T) {
	e := world(t, nil)
	e.Task = view.Task{
		ID:      "ACME-7",
		Repo:    "acme",
		Cost:    4.03,
		Started: ago(40 * time.Minute),
		Since:   ago(time.Minute),
	}

	drawn := ansi.Strip(strings.Join(e.vitals(e.Task, 100), "\n"))

	if strings.Contains(drawn, "1m") {
		t.Errorf("the duration card counts from the last state change:\n%s", drawn)
	}

	if !strings.Contains(drawn, "40m") {
		t.Errorf("the duration card does not read 40m, which is how long the run has been going:\n%s", drawn)
	}

	// And the cost it sits beside is still the whole run's.
	if !strings.Contains(drawn, "$4.03") {
		t.Errorf("the cost is gone from the row:\n%s", drawn)
	}
}
