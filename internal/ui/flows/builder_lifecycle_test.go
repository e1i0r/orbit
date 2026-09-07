package flows

import (
	"testing"

	"charm.land/bubbletea/v2"
)

func TestFlowsBuilderFullLifecycle(t *testing.T) {
	s, e := designer(t)

	// 1. The designer, open on the list.
	if s.Creating() {
		t.Fatal("the designer opened in a form rather than on the list")
	}

	sendKey := func(k rune, code rune, text string) {
		var msg tea.KeyPressMsg

		switch {
		case text != "":
			msg = tea.KeyPressMsg{Code: code, Text: text}
		case code != 0:
			msg = tea.KeyPressMsg{Code: code}
		default:
			msg = tea.KeyPressMsg{Code: k, Text: string(k)}
		}

		next, _ := s.Key(msg, e)
		s = next
	}

	// 2. Press 'n' to enter flow builder (creating = true)
	sendKey('n', 0, "")

	if !s.Creating() {
		t.Error("n did not open the form")
	}

	// A new flow opens on the tab that writes one from a sentence; the rest
	// of this walk is about the fields.
	s = s.OnFields()

	// 3. Render builder view
	if rows := s.View(e.Frame.Body.H, e.Frame.Body.W, e); len(rows) == 0 {
		t.Error("the form drew nothing")
	}

	// 4. Navigate through all builder fields with Tab
	for range flowFieldCount + 2 {
		sendKey(0, tea.KeyTab, "tab")
	}

	// 5. Navigate backwards with Shift-Tab
	for range 3 {
		sendKey(0, tea.KeyTab, "shift+tab")
	}

	// 6. Left / Right on fields to cycle options
	s.field = flowFieldTemplate

	sendKey(0, tea.KeyRight, "")
	sendKey(0, tea.KeyLeft, "")

	s.field = flowFieldEngine

	sendKey(0, tea.KeyRight, "")
	sendKey(0, tea.KeyLeft, "")

	s.field = flowFieldFeedOutput

	sendKey(0, tea.KeyRight, "")

	s.field = flowFieldWait

	sendKey(0, tea.KeyRight, "")

	// 7. Add a phase
	s.field = flowFieldAddPhase

	sendKey(0, tea.KeyEnter, "")

	if len(s.phases) < 2 {
		t.Errorf("expected at least 2 phases after AddPhase, got %d", len(s.phases))
	}

	// 8. Delete a phase
	s.field = flowFieldDelPhase

	sendKey(0, tea.KeyEnter, "")

	// 9. Save flow
	s.field = flowFieldSave
	s.flowName = "my-custom-test-flow"

	sendKey(0, tea.KeyEnter, "")

	// 10. Escape to close
	sendKey(0, tea.KeyEsc, "esc")
}

// TestFlowsBuilderRowsEditingWithPromptAndToggles renders the builder in
// the shapes TestFlowsBuilderFullLifecycle never does: editing an existing
// flow rather than creating one, a phase with a prompt long enough to
// wrap, and both toggles (feed output, wait for human) turned on.
func TestFlowsBuilderRowsEditingWithPromptAndToggles(t *testing.T) {
	s, e := designer(t)
	m2, _ := s.editFlow("careful", e)
	m2.phases[0].Prompt = "a prompt long enough that it should wrap across more than one line of the box"
	m2.phases[0].FeedOutput = true
	m2.phases[0].Wait = true
	m2.field = flowFieldPrompt

	rows := linesOf(m2.builderView(e.Frame.Body.H, e.Frame.Body.W, e))
	if len(rows) == 0 {
		t.Fatal("expected a non-empty render")
	}

	if drawn := m2.View(e.Frame.Body.H, e.Frame.Body.W, e); len(drawn) == 0 {
		t.Error("the form drew nothing while editing an existing flow")
	}
}
