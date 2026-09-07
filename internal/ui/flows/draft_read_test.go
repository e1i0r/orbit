package flows

// What the tab asks for, and what the tab does with a paragraph.
//
// Getting a flow back out of a model's answer is internal/flow's own: see
// its draft_test.go.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/prompt"
)

// TestThePromptCarriesAWholeExample. A list of field names is something a
// model improvises around; one whole document of the right shape is
// something it copies — so the example is the specification, and it has to
// be a flow this build would actually accept.
func TestThePromptCarriesAWholeExample(t *testing.T) {
	asked := prompt.FlowDraft("un loop hasta que pasen las pruebas", []string{"agy", "claude"})

	fl, err := decodeDraft(asked)
	if err != nil {
		t.Fatalf("the example this build hands out is not a flow it would accept: %v", err)
	}

	if len(fl.Phases) < 2 {
		t.Errorf("the example shows %d phases, want a loop among them", len(fl.Phases))
	}

	// On an engine this build has, and not on one it does not.
	if fl.Phases[0].Engine != "agy" {
		t.Errorf("the example runs on %q, want this build's first engine", fl.Phases[0].Engine)
	}

	// And it says which engines there are, so nothing else is invented.
	if !strings.Contains(asked, "agy, claude") {
		t.Error("the prompt does not list this build's engines")
	}
}

// TestTheSayTabTakesAParagraph, which is what somebody describing a flow
// writes.
func TestTheSayTabTakesAParagraph(t *testing.T) {
	s, e := editing(t)
	s.tab = flowTabSay

	for _, k := range []tea.KeyPressMsg{
		{Text: "a"},
		{Code: tea.KeyEnter, Mod: tea.ModShift},
		{Text: "b"},
	} {
		next, _ := s.flowsFormKey(k, e)
		s = next
	}

	if s.say != "a\nb" {
		t.Errorf("the tab holds %q", s.say)
	}
}
