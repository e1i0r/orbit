package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestAllTaskDetailPanesRender(t *testing.T) {
	m := openOn(t, "ACME-2662")

	// 1. Overview Lines
	if lines := m.overviewLines(); len(lines) == 0 {
		t.Error("overviewLines returned empty")
	}

	// 2. Flow Lines
	if lines := m.flowLines(); len(lines) == 0 {
		t.Error("flowLines returned empty")
	}

	// 3. Gates Lines
	if lines := m.gatesLines(); len(lines) == 0 {
		t.Error("gatesLines returned empty")
	}

	// 4. Cost Lines
	if lines := m.costLines(); len(lines) == 0 {
		t.Error("costLines returned empty")
	}

	// 5. Refused Lines
	if lines := m.refusedLines(); len(lines) == 0 {
		t.Error("refusedLines returned empty")
	}

	// 6. Thinking Lines
	if lines := m.thinkingLines(); len(lines) == 0 {
		t.Error("thinkingLines returned empty")
	}

	// 7. Notes Lines
	if lines := m.notesLines(); len(lines) == 0 {
		t.Error("notesLines returned empty")
	}

	// 8. Artifacts Lines
	if lines := m.artifactsLines(); len(lines) == 0 {
		t.Error("artifactsLines returned empty")
	}

	// 9. Report Lines
	if lines := m.reportLines(); len(lines) == 0 {
		t.Error("reportLines returned empty")
	}

	// 10. Log/Timeline Lines
	if lines := m.logLines(); len(lines) == 0 {
		t.Error("logLines returned empty")
	}

	// 11. Diff Lines
	if lines := m.diffLines(); len(lines) == 0 {
		t.Error("diffLines returned empty")
	}
}

func TestDetailTabsCyclingAndKeys(t *testing.T) {
	m := openOn(t, "ACME-2662")

	// Select tab directly by index and render body
	for i := 0; i < int(tabCount); i++ {
		m.tab = tab(i)

		rendered := m.detailRows(20, 100)
		if len(rendered) == 0 {
			t.Errorf("detailRows on tab %v returned empty output", i)
		}
	}
}

// TestALongPaneDrawsItsBar. The line at the foot of the pane says the arrows
// scroll; it does not say how far down eleven screens of log the reader is.
// The bar is drawn in the pane's own last column, on every row of it.
func TestALongPaneDrawsItsBar(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", longLog())
	m = showing(t, m, tabTimeline)

	const h, w = 12, 100

	rows := m.paneRows(h, w)
	if len(rows) != h {
		t.Fatalf("paneRows drew %d rows, want %d", len(rows), h)
	}

	for i, r := range rows {
		if got := lipgloss.Width(r); got != w {
			t.Errorf("row %d is %d cells wide, want the pane's %d", i, got, w)
		}

		if !strings.HasSuffix(ansi.Strip(r), scrollRail) && !strings.HasSuffix(ansi.Strip(r), scrollThumb) {
			t.Errorf("row %d ends without the bar: %q", i, ansi.Strip(r))
		}
	}
}
