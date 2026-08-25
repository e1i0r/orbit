package layout

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

func TestColumnAtBoundsAndGaps(t *testing.T) {
	plan := Plan{
		Repo:    10,
		ID:      8,
		Title:   30,
		State:   12,
		Model:   8,
		Elapsed: 7,
	}

	// 1. Negative x
	if col, ok := plan.ColumnAt(-1); ok || col != ColumnNone {
		t.Errorf("ColumnAt(-1) = (%v, %v), want (ColumnNone, false)", col, ok)
	}

	// 2. Repo column (0 to 9)
	if col, ok := plan.ColumnAt(0); !ok || col != ColumnRepo {
		t.Errorf("ColumnAt(0) = (%v, %v), want (ColumnRepo, true)", col, ok)
	}
	if col, ok := plan.ColumnAt(9); !ok || col != ColumnRepo {
		t.Errorf("ColumnAt(9) = (%v, %v), want (ColumnRepo, true)", col, ok)
	}

	// 3. Gap between Repo and ID (10 and 11)
	if col, ok := plan.ColumnAt(10); ok || col != ColumnNone {
		t.Errorf("ColumnAt(10) in gap = (%v, %v), want (ColumnNone, false)", col, ok)
	}
	if col, ok := plan.ColumnAt(11); ok || col != ColumnNone {
		t.Errorf("ColumnAt(11) in gap = (%v, %v), want (ColumnNone, false)", col, ok)
	}

	// 4. ID column (12 to 19)
	if col, ok := plan.ColumnAt(12); !ok || col != ColumnID {
		t.Errorf("ColumnAt(12) = (%v, %v), want (ColumnID, true)", col, ok)
	}

	// 5. Past the end
	if col, ok := plan.ColumnAt(500); ok || col != ColumnNone {
		t.Errorf("ColumnAt(500) past end = (%v, %v), want (ColumnNone, false)", col, ok)
	}
}

func TestTooNarrowErrorFormatting(t *testing.T) {
	err := TooNarrowError{Need: 60, Got: 45}
	if !strings.Contains(err.Error(), "60") || !strings.Contains(err.Error(), "45") {
		t.Errorf("unexpected error text: %q", err.Error())
	}
}

func FuzzFitLayout(f *testing.F) {
	f.Add(80, 24)
	f.Add(60, 10)
	f.Add(120, 40)
	f.Add(0, 0)
	f.Add(-10, -5)
	f.Add(500, 200)

	f.Fuzz(func(t *testing.T, w, h int) {
		frame, err := Fit(w, h)
		if err == nil {
			if w < MinWidth {
				t.Errorf("Fit accepted w=%d < MinWidth %d", w, MinWidth)
			}
			// Verify non-negative heights and tiling
			totH := frame.Header.H + frame.Status.H + frame.Body.H + frame.Band.H + frame.Bar.H
			if totH != h {
				t.Errorf("Fit(%d, %d) total height %d != %d", w, h, totH, h)
			}
			if frame.Header.H < 0 || frame.Status.H < 0 || frame.Body.H < 0 || frame.Band.H < 0 || frame.Bar.H < 0 {
				t.Errorf("Fit(%d, %d) produced negative height: %+v", w, h, frame)
			}
		}
	})
}

func FuzzColumnPlan(f *testing.F) {
	f.Add(80, "repo1", "ACME-1", "Task Title", "claude")
	f.Add(60, "repo1", "ID", "Short", "")
	f.Add(140, "long-repo-name", "VERY-LONG-ID-1234", "Very detailed title describing the task work", "sonnet")

	f.Fuzz(func(t *testing.T, w int, r, id, title, model string) {
		tasks := []view.Task{
			{Repo: r, ID: id, Title: title, Model: model},
		}
		p := Columns(w, tasks, func(k string) int { return 10 })
		if p.Width() > w && !p.Fallback && w >= MinWidth {
			t.Errorf("Columns width %d > %d", p.Width(), w)
		}
	})
}
