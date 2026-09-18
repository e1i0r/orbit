package flow

// Reading a flow out of what a model printed: prose around it, a second
// object beside it, a fence over it, and the writer's own line breaks inside
// its strings.

import (
	"strings"
	"testing"
)

// TestADraftWithNoFlowInItSaysSo rather than leaving the reader with an
// empty form and no reason for it.
func TestADraftWithNoFlowInItSaysSo(t *testing.T) {
	if _, err := Draft("I cannot do that"); err == nil {
		t.Error("prose was read as a flow")
	}

	fl, err := Draft(`{"phases":[{"name":"one","engine":"claude"}]}`)
	if err != nil {
		t.Fatalf("a flow with no name of its own was refused: %v", err)
	}

	if fl.Name != "draft" {
		t.Errorf("the unnamed draft is called %q", fl.Name)
	}
}

// TestADraftWithRealNewlinesInItIsMended. A model asked for JSON writes a
// prompt with line breaks and leaves them raw inside the string, which is
// not JSON — and the reader was handed "invalid character in string literal"
// instead of the flow they asked for.
func TestADraftWithRealNewlinesInItIsMended(t *testing.T) {
	raw := "{\"name\":\"mended\",\"phases\":[{\"name\":\"one\",\"engine\":\"claude\"," +
		"\"prompt\":\"first line\nsecond line\"}]}"

	fl, err := Draft(raw)
	if err != nil {
		t.Fatalf("a draft with a real newline in it was refused: %v", err)
	}

	if got := fl.Phases[0].Prompt; got != "first line\nsecond line" {
		t.Errorf("the prompt came back as %q", got)
	}

	// What was already valid is untouched, escapes included.
	same := `{"name":"same","phases":[{"name":"one","engine":"claude","prompt":"a \"quoted\" word\nand a line"}]}`

	fl, err = Draft(same)
	if err != nil {
		t.Fatalf("a valid draft was refused: %v", err)
	}

	if got := fl.Phases[0].Prompt; got != "a \"quoted\" word\nand a line" {
		t.Errorf("the valid prompt came back as %q", got)
	}
}

// TestTheFlowIsFoundAmongWhateverElseWasPrinted. An engine explains itself
// before and after the document, prints a second object to illustrate a
// point, and wraps the lot in a fence — and the reader was handed a decoder
// error about a field they never typed.
func TestTheFlowIsFoundAmongWhateverElseWasPrinted(t *testing.T) {
	answer := "Here is the flow you asked for:\n\n```json\n" +
		`{"note":"this one is an example of a phase"}` + "\n" +
		`{"name":"found","phases":[{"name":"one","engine":"claude","prompt":"do it"}]}` +
		"\n```\n\nIt has one phase. Set wait: true if you want it to stop.\n"

	fl, err := Draft(answer)
	if err != nil {
		t.Fatalf("the flow was not found: %v", err)
	}

	if fl.Name != "found" || len(fl.Phases) != 1 {
		t.Errorf("read %q with %d phases", fl.Name, len(fl.Phases))
	}
}

// TestABraceInsideAPromptOpensNothing, so a document is not cut short by
// the instructions written into it.
func TestABraceInsideAPromptOpensNothing(t *testing.T) {
	answer := `{"name":"braces","phases":[{"name":"one","engine":"claude",` +
		`"prompt":"write func f() { return } and keep going"}]}`

	fl, err := Draft(answer)
	if err != nil {
		t.Fatalf("a prompt with braces in it broke the read: %v", err)
	}

	if got := fl.Phases[0].Prompt; got != "write func f() { return } and keep going" {
		t.Errorf("the prompt came back as %q", got)
	}
}

// TestAFenceIsTakenOffWhateverIsInsideIt, which is what an engine writes on
// the days it ignores being asked not to.
func TestAFenceIsTakenOffWhateverIsInsideIt(t *testing.T) {
	const doc = `{"name":"drafted"}`

	for _, c := range []struct{ what, in string }{
		{"a fence with a language on it", "```json\n" + doc + "\n```"},
		{"a bare fence", "```\n" + doc + "\n```"},
		{"a fence with prose around it", "Here you go:\n\n```json\n" + doc + "\n```\n"},
	} {
		if got := fenced(c.in); !strings.Contains(got, doc) {
			t.Errorf("%s: fenced() = %q, want the document inside it", c.what, got)
		}
	}

	// What was never fenced is handed back exactly as it came.
	if got := fenced(doc); got != doc {
		t.Errorf("an unfenced answer came back as %q", got)
	}

	// A fence nobody closed is not a fence to take off: the answer is
	// whatever it is, and cutting the first line off it would lose part of
	// what the engine said.
	half := "```json"
	if got := fenced(half); got != half {
		t.Errorf("an unclosed fence came back as %q", got)
	}
}

// TestAClosingBraceBeforeAnyOpeningOneIsNotDepth.
//
// The reading counts braces to find where an object starts and ends. A
// model that printed prose with a stray } in it — the end of a sentence
// about a template, say — would take the count below zero, and every object
// after it would be read from the wrong place.
func TestAClosingBraceBeforeAnyOpeningOneIsNotDepth(t *testing.T) {
	out := `} here is the flow you asked for:
{"name":"careful","phases":[{"name":"implement","engine":"claude"}]}`

	raw, held := flowJSON(out)
	if !held {
		t.Fatal("a stray closing brace lost the flow that came after it")
	}

	if !strings.Contains(raw, `"phases"`) {
		t.Errorf("it read %q, want the object with phases in it", raw)
	}

	if strings.HasPrefix(raw, "}") {
		t.Errorf("it read %q, want an object that starts where one does", raw)
	}
}

// TestAnObjectThatIsNotJSONIsNotTheFlow. The first object with phases in it
// wins, and "with phases in it" means the document parsed and had them —
// not that the characters p-h-a-s-e-s were somewhere in the braces.
func TestAnObjectThatIsNotJSONIsNotTheFlow(t *testing.T) {
	out := `{"phases": this is not json at all}
{"name":"careful","phases":[{"name":"implement","engine":"claude"}]}`

	raw, held := flowJSON(out)
	if !held {
		t.Fatal("nothing was read as a flow")
	}

	if strings.Contains(raw, "not json") {
		t.Errorf("it read %q, want the object that parses", raw)
	}
}

// TestTheLongestOfTwoEqualObjectsIsTheFirst.
//
// The fallback is the longest object, which is the best guess left when
// nothing parses. Two of the same length is a tie, and a tie broken
// differently on two runs of the same answer is a draft that cannot be
// reproduced — so it is the first, and this says so.
func TestTheLongestOfTwoEqualObjectsIsTheFirst(t *testing.T) {
	out := `{"a":"first one here!"}
{"b":"second one here"}`

	raw, held := flowJSON(out)
	if !held {
		t.Fatal("neither object came back")
	}

	if !strings.Contains(raw, `"a"`) {
		t.Errorf("it read %q, want the first of the two", raw)
	}
}

// TestAFenceWithNothingInsideItIsNothing. An engine that opened a block and
// closed it without writing anything said nothing, and a reading that
// answered with the fence itself would hand three backticks to the parser.
func TestAFenceWithNothingInsideItIsNothing(t *testing.T) {
	if got := fenced("```json\n```"); strings.Contains(got, "`") {
		t.Errorf("an empty fenced block came back as %q", got)
	}

	// And one with something in it comes back with the something.
	got := fenced("```json\n{\"name\":\"careful\"}\n```")
	if !strings.Contains(got, "careful") {
		t.Errorf("a fenced block came back as %q", got)
	}

	if strings.Contains(got, "`") {
		t.Errorf("the fence came back with it: %q", got)
	}
}

// TestALineExactlyAtTheLimitIsNotCutShort. The message says what came back
// instead of a flow without printing a page of it, and a line of exactly the
// length allowed is one that fits — cutting it would put an ellipsis on an
// answer that was already whole.
func TestALineExactlyAtTheLimitIsNotCutShort(t *testing.T) {
	const limit = 120

	exact := strings.Repeat("x", limit)
	if got := opening(exact); got != exact {
		t.Errorf("a line of exactly %d characters came back %d long", limit, len(got))
	}

	over := strings.Repeat("x", limit+1)
	if got := opening(over); !strings.HasSuffix(got, "…") {
		t.Errorf("a line of %d characters came back whole: %q", limit+1, got)
	}
}
