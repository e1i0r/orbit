package ui

// flowstpl_prompt_coverage_test.go is generatePhasePrompt's whole keyword
// table: one case per branch, whether it is reading the reader's own draft
// or falling back to the phase's name.

import (
	"strings"
	"testing"
)

func TestGeneratePhasePromptFromDraft(t *testing.T) {
	cases := []struct {
		name, input, wantSub string
	}{
		{"validate", "validate the new endpoint", "Valida exhaustivamente"},
		{"test keyword", "run the test suite", "Valida exhaustivamente"},
		{"security", "audit for security holes", "Audita rigurosamente"},
		{"refactor", "clean up this module", "Refactoriza"},
		{"fix", "fix the crash on startup", "Investiga la causa raíz"},
		{"docs", "write the readme", "documentación técnica"},
		{"default fallback", "ship the release notes", "máxima precisión técnica"},
	}
	for _, c := range cases {
		got := generatePhasePrompt(c.input, "some-phase", "some-flow")
		if !strings.Contains(got, c.wantSub) {
			t.Errorf("%s: generatePhasePrompt(%q) = %q, want it to contain %q", c.name, c.input, got, c.wantSub)
		}

		if !strings.Contains(got, c.input) {
			t.Errorf("%s: expected the draft %q to survive into the generated prompt %q", c.name, c.input, got)
		}
	}
}

func TestGeneratePhasePromptFromPhaseName(t *testing.T) {
	cases := []struct {
		phase, wantSub string
	}{
		{"1-plan", "diseña un plan técnico"},
		{"design-review", "diseña un plan técnico"},
		{"2-implement", "Implementa la solución"},
		{"build", "Implementa la solución"},
		{"3-test", "pruebas automatizadas"},
		{"qa-gate", "pruebas automatizadas"},
		{"4-review", "Audita el diff"},
		{"security-audit", "Audita el diff"},
		{"5-fix", "Corrige con precisión"},
		{"remediate", "Corrige con precisión"},
	}
	for _, c := range cases {
		got := generatePhasePrompt("", c.phase, "some-flow")
		if !strings.Contains(got, c.wantSub) {
			t.Errorf("phase %q: generatePhasePrompt = %q, want it to contain %q", c.phase, got, c.wantSub)
		}
	}
}

func TestGeneratePhasePromptDefaultFallback(t *testing.T) {
	withFlow := generatePhasePrompt("", "mystery-phase", "my-flow")
	if !strings.Contains(withFlow, "mystery-phase") || !strings.Contains(withFlow, "my-flow") {
		t.Errorf("expected both names in the fallback, got %q", withFlow)
	}

	withoutFlow := generatePhasePrompt("", "mystery-phase", "")
	if !strings.Contains(withoutFlow, "mystery-phase") {
		t.Errorf("expected the phase name in the fallback, got %q", withoutFlow)
	}

	if strings.Contains(withoutFlow, "para el flujo") {
		t.Errorf("expected no flow clause when flowName is empty, got %q", withoutFlow)
	}
}
