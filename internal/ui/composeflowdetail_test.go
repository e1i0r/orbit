package ui

// What the form says about the flow it is set to. It was one line of every
// phase and the whole description run together, which at any width ended in
// an ellipsis part-way through the sentence a reader was choosing on.

import (
	"strings"
	"testing"
)

// TestTheChosenFlowIsReadAsAListAndNotAParagraph. A flow is a sequence of
// phases, and a sequence set as prose is a sequence nobody counts: the rows
// are the phases, in the order they run, numbered.
func TestTheChosenFlowIsReadAsAListAndNotAParagraph(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	rows := m.flowDetail("careful", 120)
	if len(rows) < 4 {
		t.Fatalf("the flow detail is %d rows, want a row per phase and its description: %v", len(rows), rows)
	}

	for i, want := range []string{"1 implement", "2 review", "3 fix"} {
		if !strings.HasPrefix(rows[i], want) {
			t.Errorf("row %d is %q, want it to start %q", i, rows[i], want)
		}

		if strings.Contains(rows[i], "Thorough") {
			t.Errorf("row %d carries the description as well as the phase: %q", i, rows[i])
		}
	}

	// The pause is the one thing about a flow that surprises whoever chose
	// it, so it is a phrase and not only a glyph a reader has to know.
	if !strings.Contains(rows[1], "waits for you") {
		t.Errorf("the phase that stops does not say so: %q", rows[1])
	}

	if !strings.Contains(strings.Join(rows[3:], " "), "Thorough multiphase") {
		t.Errorf("the flow's own description is not under the phases: %v", rows[3:])
	}
}

// TestALongDescriptionStopsRatherThanPushingTheFormDown. The task box is the
// field a reader is on their way to, and prose somebody wrote about their own
// flow must not walk it off the bottom of a short terminal.
func TestALongDescriptionStopsRatherThanPushingTheFormDown(t *testing.T) {
	m, _ := testModel(t, 60, 30)

	rows := m.flowDetail("careful", 60)
	if len(rows) != 3+flowAboutRows {
		t.Fatalf("a narrow form draws %d rows, want the phases and %d of description: %v",
			len(rows), flowAboutRows, rows)
	}

	if last := rows[len(rows)-1]; !strings.HasSuffix(last, "…") {
		t.Errorf("the description was cut without saying so: %q", last)
	}
}

// TestPointingAnywhereInTheBlockOpensTheFlow. It was one row and is now
// several; a reader aiming at the sentence that made them curious lands on
// the row that answers.
func TestPointingAnywhereInTheBlockOpensTheFlow(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m = m.openCompose()

	plan := m.composeLayout()
	if plan.flowRows < 2 {
		t.Fatalf("the flow detail is %d rows, want several to point at", plan.flowRows)
	}

	y := m.frame.Body.Y + plan.flowSum + plan.flowRows - 1
	if got := m.hit(20, y); got.Kind != TargetComposeInspectFlow {
		t.Errorf("the last row of the block is kind %d, want the flow inspector", got.Kind)
	}
}
