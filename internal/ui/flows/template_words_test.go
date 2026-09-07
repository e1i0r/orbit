package flows

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
	base, e := designer(t)
	base = base.startCreateFlow(e).OnFields()

	for _, tpl := range []string{"TDD Fuzz & PR", "TDD Cycle", "Security Audit", "Turbo Fix", "ninguna"} {
		s := base
		s.activePhase = 2

		s, out1 := s.applyFlowTemplate(tpl, e)
		if out1.Said == "a sentence from before" {
			t.Errorf("applyFlowTemplate(%q) said nothing; the band still reads %q", tpl, out1.Said)
		}

		if s.activePhase != 0 {
			t.Errorf("applyFlowTemplate(%q) left the cursor on phase %d, want the first", tpl, s.activePhase)
		}
	}
}

// TestTheBuilderSpeaksTheReadersLanguage. Six of these sentences were
// Spanish string literals in the middle of a screen whose every other line
// comes out of the catalogue, so an English reader was told "plantilla TDD
// Cycle cargada (3 fases)" and a Spanish one could not have them retranslated.
func TestTheBuilderSpeaksTheReadersLanguage(t *testing.T) {
	s, e := designer(t)
	s = s.startCreateFlow(e).OnFields()

	for _, tpl := range []string{"TDD Fuzz & PR", "TDD Cycle", "Security Audit", "Turbo Fix", "ninguna"} {
		_, out2 := s.applyFlowTemplate(tpl, e)
		for _, spanish := range []string{"plantilla", "cargada", "fases", "flujo en blanco"} {
			if strings.Contains(out2.Said, spanish) {
				t.Errorf("applyFlowTemplate(%q) said %q, which is Spanish written into the code", tpl, out2.Said)
			}
		}
	}

	blank := s
	blank.flowName = "  "

	_, out3 := blank.saveCustomFlow(e)
	if strings.Contains(out3.Said, "indica un nombre") {
		t.Errorf("saveCustomFlow refused in Spanish written into the code: %q", out3.Said)
	}
}

// TestAPastedPromptIsCountedInCharacters. The count was len(), which is
// bytes: an accented word is reported longer than it is and one emoji as
// four characters. Nobody counts a pasted prompt to check the number, which
// is the whole reason it has to be right.
func TestAPastedPromptIsCountedInCharacters(t *testing.T) {
	s, e := designer(t)
	s = s.startCreateFlow(e).OnFields()

	// Three characters, eleven bytes.
	got, gotOut := s.pastedPrompt("áé🎉", e)
	if !strings.Contains(gotOut.Said, "3 ") {
		t.Errorf("pasting three characters said %q, want it to count 3", gotOut.Said)
	}

	if strings.Contains(gotOut.Said, "11") {
		t.Errorf("pasting three characters counted its bytes: %q", gotOut.Said)
	}

	if got.cur().Prompt != "áé🎉" {
		t.Errorf("the paste landed %q in the phase", got.cur().Prompt)
	}

	// Nothing on the clipboard says so rather than reporting a paste of
	// nothing into the phase.
	_, emptyOut := s.pastedPrompt("", e)
	if !strings.Contains(emptyOut.Said, "clipboard") {
		t.Errorf("pasting nothing said %q, want it to name the clipboard", emptyOut.Said)
	}
}
