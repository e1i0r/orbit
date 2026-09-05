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

// TestAnAnswerThatIsNotJSONIsSentBack. A model writing JSON by hand puts a
// quotation mark inside a string — "go test ./..." is what the person asked
// for and the engine repeats it — and from there no repair here can tell a
// string that ran on from one that was never closed. The engine that wrote
// it is asked, once, with the decoder's own complaint.
func TestAnAnswerThatIsNotJSONIsSentBack(t *testing.T) {
	m := builderModel(t)
	m.flows.tab = flowTabSay
	m.flows.say = "un loop hasta que pasen las pruebas"

	asks := 0
	second := ""

	m.opts.Draft = func(engineName, model, prompt string) (string, error) {
		asks++
		if asks == 1 {
			// A quote inside a string, exactly as it comes back.
			return `{"name":"safe-refactor","phases":[{"name":"one","engine":"claude",` +
				`"prompt":"run "go test ./..." until it passes"}]}`, nil
		}

		second = prompt

		return `{"name":"safe-refactor","phases":[{"name":"one","engine":"claude",` +
			`"prompt":"run go test ./... until it passes"}]}`, nil
	}

	_, cmd := m.draftFlow()

	answer := draftAnswer(t, cmd)
	if answer.err != nil {
		t.Fatalf("the mended draft was refused: %v", answer.err)
	}

	if asks != 2 {
		t.Errorf("the engine was asked %d times, want a first answer and one mend", asks)
	}

	if !answer.mended {
		t.Error("the draft does not say it took two asks")
	}

	if !strings.Contains(second, "not valid JSON") || !strings.Contains(second, "go test") {
		t.Errorf("the second ask does not carry the complaint and what was written:\n%s", second)
	}

	// And a first answer that parses is never sent back.
	asks = 0
	m.opts.Draft = func(engineName, model, prompt string) (string, error) {
		asks++

		return `{"name":"fine","phases":[{"name":"one","engine":"claude"}]}`, nil
	}

	_, cmd = m.draftFlow()

	if got := draftAnswer(t, cmd); got.mended || asks != 1 {
		t.Errorf("a valid answer was asked for again: asks=%d mended=%v", asks, got.mended)
	}
}

// TestTheDraftSaysWhatItIsDoing, in the bar, both when it is sent and when
// it comes back with nothing. A gesture whose effect cannot be seen is one
// the reader presses again — and behind this one is a run they pay for.
func TestTheDraftSaysWhatItIsDoing(t *testing.T) {
	m := builderModel(t)
	m.flows.tab = flowTabSay
	m.flows.say = "implementa y revisa"
	m.opts.Draft = func(engineName, model, prompt string) (string, error) { return "{}", nil }

	sent, _ := m.draftFlow()
	if sent.message == "" {
		t.Error("nothing was said in the bar when the draft was sent")
	}

	if !sent.flows.saying {
		t.Error("the tab does not say it is waiting on the engine")
	}

	back, _ := sent.drafted(flowDraftedMsg{id: sent.flows.sayID, err: errDraftForTest})

	got := asModel(t, back)
	if got.message == "" || got.flows.saying {
		t.Errorf("a refused draft left the bar at %q and saying=%v", got.message, got.flows.saying)
	}
}
