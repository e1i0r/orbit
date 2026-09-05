package ui

// Reading what an engine wrote back: what is a flow, what is prose, and what
// is JSON with the writer's own quotation marks left in it.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestADraftWithNoFlowInItSaysSo rather than leaving the reader with an
// empty form and no reason for it.
func TestADraftWithNoFlowInItSaysSo(t *testing.T) {
	if _, err := decodeDraft("I cannot do that"); err == nil {
		t.Error("prose was read as a flow")
	}

	fl, err := decodeDraft(`{"phases":[{"name":"one","engine":"claude"}]}`)
	if err != nil {
		t.Fatalf("a flow with no name of its own was refused: %v", err)
	}

	if fl.Name != "draft" {
		t.Errorf("the unnamed draft is called %q", fl.Name)
	}
}

// TestTheSayTabTakesAParagraph, which is what somebody describing a flow
// writes.
func TestTheSayTabTakesAParagraph(t *testing.T) {
	m := builderModel(t)
	m.flows.tab = flowTabSay

	for _, k := range []tea.KeyPressMsg{
		{Text: "a"},
		{Code: tea.KeyEnter, Mod: tea.ModShift},
		{Text: "b"},
	} {
		next, _ := m.flowsFormKey(k)
		m = asModel(t, next)
	}

	if m.flows.say != "a\nb" {
		t.Errorf("the tab holds %q", m.flows.say)
	}
}

// TestTheDraftButtonIsOne. It is drawn as a pill, so it has to answer to
// the pointer: a button that only works from the keyboard is a picture of a
// button.
func TestTheDraftButtonIsOne(t *testing.T) {
	m := builderModel(t)
	m.flows.tab = flowTabSay
	m.flows.say = "implementa y revisa"

	asked := false
	m.opts.Draft = func(engineName, model, prompt string) (string, error) {
		asked = true

		return "{}", nil
	}

	lines, start := m.builderView(m.frame.Body.H, m.frame.Body.W)

	y := -1

	for i, l := range lines {
		if l.act == "draft" {
			y = m.frame.Body.Y + i - start
		}
	}

	if y < 0 {
		t.Fatal("the tab draws no draft button")
	}

	got := m.hitFlows(4, y)
	if got.Field != "draft" {
		t.Fatalf("hitFlows on the button = %+v", got)
	}

	next, cmd := m.handleFlowClick(got)
	if cmd == nil {
		t.Fatal("clicking the button asked nothing")
	}

	if !asModel(t, next).flows.saying {
		t.Error("the click did not say it was waiting on the engine")
	}

	draftAnswer(t, cmd)

	if !asked {
		t.Error("the engine was never asked")
	}
}

// TestTheDraftSaysWhatItIsDoing, in the bar, both when it is sent and when
// it comes back with nothing.
func TestTheDraftSaysWhatItIsDoing(t *testing.T) {
	m := builderModel(t)
	m.flows.tab = flowTabSay
	m.flows.say = "implementa y revisa"
	m.opts.Draft = func(engineName, model, prompt string) (string, error) { return "{}", nil }

	sent, _ := m.draftFlow()
	if sent.message == "" {
		t.Error("nothing was said in the bar when the draft was sent")
	}

	back, _ := sent.drafted(flowDraftedMsg{id: sent.flows.sayID, err: errDraftForTest})
	if got := asModel(t, back); got.message == "" || got.flows.saying {
		t.Errorf("a refused draft left the bar at %q and saying=%v", got.message, got.flows.saying)
	}
}

// TestTheDraftIsAskedOfTheEngineTheReaderChose, which is not always the
// window's default: the one that is set up on this machine is the one that
// can answer.
func TestTheDraftIsAskedOfTheEngineTheReaderChose(t *testing.T) {
	m := builderModel(t)
	m.flows.tab = flowTabSay
	m.flows.say = "implementa y revisa"

	names := m.engineNames()
	if len(names) < 2 {
		t.Skip("this build has one engine")
	}

	next, _ := m.flowsFormKey(tea.KeyPressMsg{Code: tea.KeyRight})

	m = asModel(t, next)
	if m.sayEngineName() == names[0] {
		t.Fatalf("→ left the engine on %q", m.sayEngineName())
	}

	if m.message == "" {
		t.Error("the bar did not say which engine the draft would be asked of")
	}

	asked := ""
	m.opts.Draft = func(engineName, model, prompt string) (string, error) {
		asked = engineName

		return "{}", nil
	}

	_, cmd := m.draftFlow()
	draftAnswer(t, cmd)

	if asked != m.sayEngineName() {
		t.Errorf("the draft was asked of %q, want %q", asked, m.sayEngineName())
	}
}

// TestANewFlowOpensOnTheTabThatWritesIt, and an existing one does not: the
// sentence replaces every phase, which is not what somebody who pressed
// edit came to do.
func TestANewFlowOpensOnTheTabThatWritesIt(t *testing.T) {
	fresh, _ := testModel(t, 100, 36)

	m := fresh.startCreateFlow()
	if m.flows.tab != flowTabSay {
		t.Errorf("a new flow opened on tab %d", m.flows.tab)
	}

	next, _ := m.editNamedFlow("careful")

	editing := asModel(t, next)
	if editing.flows.tab != flowTabFields {
		t.Errorf("editing careful opened on tab %d", editing.flows.tab)
	}

	// And that tab warns that a draft would take the place of what is there.
	editing.flows.tab = flowTabSay

	rows := strings.Join(editing.flowsBuilderRows(editing.frame.Body.H, editing.frame.Body.W), "\n")
	if !strings.Contains(rows, "3") {
		t.Errorf("the tab does not say how many phases a draft would replace:\n%s", rows)
	}
}

// TestADraftWithRealNewlinesInItIsMended. A model asked for JSON writes a
// prompt with line breaks and leaves them raw inside the string, which is
// not JSON — and the reader was handed "invalid character '\n' in string
// literal" instead of the flow they asked for.
func TestADraftWithRealNewlinesInItIsMended(t *testing.T) {
	raw := "{\"name\":\"mended\",\"phases\":[{\"name\":\"one\",\"engine\":\"claude\"," +
		"\"prompt\":\"first line\nsecond line\"}]}"

	fl, err := decodeDraft(raw)
	if err != nil {
		t.Fatalf("a draft with a real newline in it was refused: %v", err)
	}

	if got := fl.Phases[0].Prompt; got != "first line\nsecond line" {
		t.Errorf("the prompt came back as %q", got)
	}

	// What was already valid is untouched, escapes included.
	same := `{"name":"same","phases":[{"name":"one","engine":"claude","prompt":"a \"quoted\" word\nand a line"}]}`

	fl, err = decodeDraft(same)
	if err != nil {
		t.Fatalf("a valid draft was refused: %v", err)
	}

	if got := fl.Phases[0].Prompt; got != "a \"quoted\" word\nand a line" {
		t.Errorf("the valid prompt came back as %q", got)
	}
}
