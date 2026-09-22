package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// TestTheSameReadingDoesNotRebuildThePanes. The record is read every half
// second, and rebuilding fifteen panes from an answer that says nothing new
// is what made the window heavy while a task ran.
func TestTheSameReadingDoesNotRebuildThePanes(t *testing.T) {
	at := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	log := []view.Entry{
		{At: at, Kind: "task.started"},
		{At: at.Add(time.Second), Kind: "phase.started", Phase: "fix"},
	}

	if !sameLog(log, append([]view.Entry(nil), log...)) {
		t.Error("two readings of the same record are said to differ")
	}

	grown := append(append([]view.Entry(nil), log...),
		view.Entry{At: at.Add(2 * time.Second), Kind: "phase.finished"})
	if sameLog(log, grown) {
		t.Error("a record that grew is said to be the same")
	}

	m, _ := testModel(t, 100, 30)
	m.detail = "ACME-1"
	m.entries = log

	// A pane sentinel: syncPanes rewrites every pane, so a pane that still
	// holds it was not rebuilt.
	m.panes[tabTimeline].SetContent("untouched")

	next, _ := m.Update(logMsg{ID: "ACME-1", Entries: append([]view.Entry(nil), log...)})
	if got := asModel(t, next).panes[tabTimeline].View(); !strings.HasPrefix(got, "untouched") {
		t.Errorf("the same record rebuilt the panes: %q", got)
	}
}
