package ui

import (
	"testing"
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
