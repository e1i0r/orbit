package ui

// The tab that turns a sentence into a flow: what it asks, what it does with
// the answer, and what happens to an answer nobody is waiting for any more.

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

// errDraftForTest is what an engine that could not answer looks like here.
var errDraftForTest = errors.New("agy: no quota source")

// draftAnswer runs what draftFlow handed back and digs the engine's answer
// out of it. What comes back is a batch — the question and the first frame
// of the spinner beside it — so a test that called the command straight got
// a batch message and never saw the flow.
func draftAnswer(t *testing.T, cmd tea.Cmd) flowDraftedMsg {
	t.Helper()

	if cmd == nil {
		t.Fatal("nothing was sent to the engine")
	}

	switch msg := cmd().(type) {
	case flowDraftedMsg:
		return msg
	case tea.BatchMsg:
		for _, one := range msg {
			if drafted, ok := one().(flowDraftedMsg); ok {
				return drafted
			}
		}
	}

	t.Fatalf("the engine's answer came back as %T", cmd())

	return flowDraftedMsg{}
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
	m.opts.Draft = func(engineName, model, prompt string) (string, error) {
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

	msg := draftAnswer(t, cmd)

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

// TestTheDraftPicksProviderAndModel. The engine that runs tasks here is not
// always the one set up to answer a question, and a model is one engine's
// own name for it.
func TestTheDraftPicksProviderAndModel(t *testing.T) {
	m := builderModel(t)
	m.flows.tab = flowTabSay
	m.flows.say = "implementa y revisa"
	m.flows.sayFocus = sayOnModel

	mdls, _ := m.modelsFor(m.sayEngineName())
	if len(mdls) == 0 {
		t.Skip("this build's engine table has no models")
	}

	m = m.turnSayDial(1)
	if m.flows.sayModel != mdls[0] {
		t.Fatalf("→ on the model dial left it at %q", m.flows.sayModel)
	}

	askedEngine, askedModel := "", ""
	m.opts.Draft = func(engineName, model, prompt string) (string, error) {
		askedEngine, askedModel = engineName, model

		return "{}", nil
	}

	_, cmd := m.draftFlow()
	draftAnswer(t, cmd)

	if askedEngine != m.sayEngineName() || askedModel != mdls[0] {
		t.Errorf("asked %q/%q, want %q/%q", askedEngine, askedModel, m.sayEngineName(), mdls[0])
	}

	// Changing the engine forgets the model, which belonged to the old one.
	m.flows.sayFocus = sayOnEngine
	m = m.turnSayDial(1)

	if m.flows.sayModel != "" {
		t.Errorf("the old engine's model survived at %q", m.flows.sayModel)
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

// TestStoppingTheWaitDropsTheAnswer. Orbit did not spawn the engine with a
// handle it can kill, so escape stops waiting rather than pretending to
// cancel — and what lands afterwards must not overwrite a form the reader
// has gone back to editing by hand.
func TestStoppingTheWaitDropsTheAnswer(t *testing.T) {
	m := builderModel(t)
	m.flows.tab = flowTabSay
	m.flows.say = "implementa y revisa"
	m.opts.Draft = func(engineName, model, prompt string) (string, error) {
		return `{"name":"late","phases":[{"name":"one","engine":"claude"}]}`, nil
	}

	sent, cmd := m.draftFlow()

	answer := draftAnswer(t, cmd)

	// Escape while the question is out.
	next, _ := sent.flowsFormKey(tea.KeyPressMsg{Code: tea.KeyEscape})

	stopped := asModel(t, next)
	if stopped.flows.saying {
		t.Fatal("escape left the tab waiting")
	}

	// The late answer lands on nothing.
	after, _ := stopped.drafted(answer)

	got := asModel(t, after)
	if got.flows.flowName == "late" || got.flows.tab != flowTabSay {
		t.Errorf("the dropped answer landed anyway: name=%q tab=%d", got.flows.flowName, got.flows.tab)
	}
}
