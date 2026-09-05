package flow

// Reading a flow out of what a model printed: prose around it, a second
// object beside it, a fence over it, and the writer's own line breaks inside
// its strings.

import "testing"

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
