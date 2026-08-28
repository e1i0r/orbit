package ui

// flowstpl_words_test.go is what the flow builder says out loud: that every
// preset says it loaded, in the reader's language, and that a pasted prompt
// is measured in characters.

import (
	"strings"
	"testing"
)

// TestEveryTemplateSaysItLoaded. Turbo Fix was the one preset that said
// nothing: pressing it left the band showing the sentence of whatever
// template had been chosen before, so the only way to tell it had loaded
// was to read the phase list.
//
// It was also the one preset that did not put the cursor back on the first
// phase. Choosing a three-phase preset and then Turbo Fix left activePhase
// at 2 in a flow with one phase — clamped on the next draw, so the tab bar
// jumped a phase on its own.
func TestEveryTemplateSaysItLoaded(t *testing.T) {
	base, _ := testModel(t, 100, 30)
	base = base.startCreateFlow()

	for _, tpl := range []string{"TDD Fuzz & PR", "TDD Cycle", "Security Audit", "Turbo Fix", "ninguna"} {
		m := base
		m.flows.activePhase = 2
		m.message = "a sentence from before"

		m, _ = m.applyFlowTemplate(tpl)
		if m.message == "a sentence from before" {
			t.Errorf("applyFlowTemplate(%q) said nothing; the band still reads %q", tpl, m.message)
		}

		if m.flows.activePhase != 0 {
			t.Errorf("applyFlowTemplate(%q) left the cursor on phase %d, want the first", tpl, m.flows.activePhase)
		}
	}
}

// TestTheBuilderSpeaksTheReadersLanguage. Six of these sentences were
// Spanish string literals in the middle of a screen whose every other line
// comes out of the catalogue, so an English reader was told "plantilla TDD
// Cycle cargada (3 fases)" and a Spanish one could not have them retranslated.
func TestTheBuilderSpeaksTheReadersLanguage(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.startCreateFlow()

	for _, tpl := range []string{"TDD Fuzz & PR", "TDD Cycle", "Security Audit", "Turbo Fix", "ninguna"} {
		got, _ := m.applyFlowTemplate(tpl)
		for _, spanish := range []string{"plantilla", "cargada", "fases", "flujo en blanco"} {
			if strings.Contains(got.message, spanish) {
				t.Errorf("applyFlowTemplate(%q) said %q, which is Spanish written into the code", tpl, got.message)
			}
		}
	}

	blank := m
	blank.flows.flowName = "  "

	refused, _ := blank.saveCustomFlow()
	if strings.Contains(refused.message, "indica un nombre") {
		t.Errorf("saveCustomFlow refused in Spanish written into the code: %q", refused.message)
	}
}

// TestAPastedPromptIsCountedInCharacters. The count was len(), which is
// bytes: an accented word is reported longer than it is and one emoji as
// four characters. Nobody counts a pasted prompt to check the number, which
// is the whole reason it has to be right.
func TestAPastedPromptIsCountedInCharacters(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.startCreateFlow()

	// Three characters, eleven bytes.
	got := m.pastedPrompt("áé🎉")
	if !strings.Contains(got.message, "3 ") {
		t.Errorf("pasting three characters said %q, want it to count 3", got.message)
	}

	if strings.Contains(got.message, "11") {
		t.Errorf("pasting three characters counted its bytes: %q", got.message)
	}

	if got.flows.cur().Prompt != "áé🎉" {
		t.Errorf("the paste landed %q in the phase", got.flows.cur().Prompt)
	}

	// Nothing on the clipboard says so rather than reporting a paste of
	// nothing into the phase.
	empty := m.pastedPrompt("")
	if !strings.Contains(empty.message, "clipboard") {
		t.Errorf("pasting nothing said %q, want it to name the clipboard", empty.message)
	}
}
