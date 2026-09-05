package ui

// The designer's three tabs.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

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
