package panes

// The run every test in this package reads: one task, the record it left,
// and a body to draw it in.

import (
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// now is the instant every elapsed column in these tests is measured
// against, so that a duration is a fact about the record and not about how
// long the suite took to get here.
var now = time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

// ago is a moment before now.
func ago(d time.Duration) time.Time { return now.Add(-d) }

// world is the Env: a run on the board, its record, and a hundred columns
// to draw it in.
func world(t *testing.T, entries []view.Entry) Env {
	t.Helper()

	frame, err := layout.Fit(100, 30)
	if err != nil {
		t.Fatalf("a hundred columns is too narrow to draw in: %v", err)
	}

	return Env{
		Words:   words.For("en"),
		Frame:   frame,
		Now:     now,
		Task:    view.Task{ID: "ACME-7", Repo: "acme", Engine: "claude", Model: "opus"},
		Entries: entries,
		Priced:  true,
	}
}
