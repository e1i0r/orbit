package ui

// The designer's three tabs.

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

// errDraftForTest is what an engine that could not answer looks like here.
var errDraftForTest = errors.New("agy: no quota source")

// TestTheTabsAreThreeViewsOfOneFlow: what the diagram draws is what the
// fields hold, without anything being copied between them.
func TestTheTabsAreThreeViewsOfOneFlow(t *testing.T) {
	m := builderModel(t)
	m.flows.phases[0].Name = "escribe-pruebas"

	m = m.moveFlowTab(1)
	if m.flows.tab != flowTabDiagram {
		t.Fatalf("^→ landed on tab %d", m.flows.tab)
	}

	rows := strings.Join(m.flowsBuilderRows(m.frame.Body.H, m.frame.Body.W), "\n")
	if !strings.Contains(rows, "escribe-pruebas") {
		t.Errorf("the diagram does not show the phase being edited:\n%s", rows)
	}

	// And round the three of them, back to the fields.
	m = m.moveFlowTab(1)
	m = m.moveFlowTab(1)

	if m.flows.tab != flowTabFields {
		t.Errorf("three moves landed on tab %d, want the fields", m.flows.tab)
	}
}

// TestClickingATabOpensIt, and clicking a phase in the diagram takes the
// reader to the fields of that phase.
func TestClickingATabOpensIt(t *testing.T) {
	m := builderModel(t)

	lines, start := m.builderView(m.frame.Body.H, m.frame.Body.W)

	y := 0

	for i, l := range lines {
		if l.strip {
			y = m.frame.Body.Y + i - start
		}
	}

	// The strip's second name is the diagram.
	x := 2 + lipgloss.Width(m.flowTabNames()[0]) + 3

	got := m.hitFlows(x, y)
	if got.Field != "tab" || got.Phase != flowTabDiagram {
		t.Fatalf("hitFlows on the second tab = %+v", got)
	}

	next, _ := m.handleFlowClick(got)

	after := asModel(t, next)
	if after.flows.tab != flowTabDiagram {
		t.Errorf("the click left tab %d open", after.flows.tab)
	}
}

// TestADraftLandsInTheFieldsAndIsNotSaved. A flow is a standing instruction
// that spends money on every task written against it, so what an engine
// wrote is read by a person before it becomes one.
func TestADraftLandsInTheFieldsAndIsNotSaved(t *testing.T) {
	m := builderModel(t)
	m.opts.Flows = flowsTestDir(t.TempDir())
	m.flows.tab = flowTabSay
	m.flows.say = "implementa y luego revisa"

	asked := ""
	m.opts.Draft = func(engineName, prompt string) (string, error) {
		asked = prompt

		return "Here you go:\n```json\n" + `{"name":"revisado","description":"dos fases",` +
			`"phases":[{"name":"1-implement","engine":"claude","prompt":"hazlo"},` +
			`{"name":"2-review","engine":"claude","wait":true,"prompt":"revisa"}]}` + "\n```\n", nil
	}

	next, cmd := m.draftFlow()
	if !next.flows.saying {
		t.Fatal("the tab does not say it is waiting on the engine")
	}

	if cmd == nil {
		t.Fatal("nothing was sent to the engine")
	}

	msg, ok := cmd().(flowDraftedMsg)
	if !ok {
		t.Fatalf("the engine's answer came back as %T", cmd())
	}

	if msg.err != nil {
		t.Fatalf("the draft was refused: %v", msg.err)
	}

	done, _ := next.drafted(msg)

	after := asModel(t, done)
	if len(after.flows.phases) != 2 || after.flows.phases[1].Name != "2-review" {
		t.Errorf("the draft did not land in the fields: %+v", after.flows.phases)
	}

	if after.flows.tab != flowTabFields {
		t.Error("the draft did not open the fields for the reader to check")
	}

	if _, err := flow.Resolve(after.opts.Flows, "revisado"); err == nil {
		t.Error("the draft saved itself without anybody reading it")
	}

	// The engine is told which engines exist, so it cannot name one this
	// machine has never had.
	if !strings.Contains(asked, strings.Join(m.engineNames(), ", ")) {
		t.Errorf("the prompt does not list this build's engines:\n%s", asked)
	}
}

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
	m.opts.Draft = func(engineName, prompt string) (string, error) {
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

	cmd()

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
	m.opts.Draft = func(engineName, prompt string) (string, error) { return "{}", nil }

	sent, _ := m.draftFlow()
	if sent.message == "" {
		t.Error("nothing was said in the bar when the draft was sent")
	}

	back, _ := sent.drafted(flowDraftedMsg{err: errDraftForTest})
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
	m.opts.Draft = func(engineName, prompt string) (string, error) {
		asked = engineName

		return "{}", nil
	}

	_, cmd := m.draftFlow()
	cmd()

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
