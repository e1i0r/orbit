package engine

// The two halves of an agreement: what every model Orbit runs is asked to
// send back, and how anything already written to that agreement is quoted
// into the next prompt without coming apart.

import (
	"strings"
	"testing"
)

// TestTheContractNamesWhatCannotBeDrawn. The rules are not a house style, so
// they are pinned here rather than only in the constant: dropping one of them
// is a change in what an engine is allowed to send back to a pane that cannot
// lay it out, and that is a failure and not an edit.
func TestTheContractNamesWhatCannotBeDrawn(t *testing.T) {
	contract := AnswerContract

	for _, want := range []string{"Markdown", "##", "###", "- ", "fence", "tables", "HTML"} {
		if !strings.Contains(contract, want) {
			t.Errorf("the contract says nothing about %q:\n%s", want, contract)
		}
	}
}

// TestAFenceOutgrowsTheFencesInsideIt. Three backticks around an answer that
// contains three backticks close on the answer's own fence, and everything
// past that point reads as prose of the prompt.
func TestAFenceOutgrowsTheFencesInsideIt(t *testing.T) {
	cases := []struct{ name, text, want string }{
		{"nothing to outgrow", "plain prose", "```"},
		{"a fenced block inside", "before\n```go\ncode\n```\nafter", "````"},
		{"a fence already grown", "before\n````\nquoted\n````\nafter", "`````"},
	}

	for _, c := range cases {
		got := Fenced(c.text)

		if !strings.HasPrefix(got, c.want+"markdown\n") || !strings.HasSuffix(got, "\n"+c.want) {
			t.Errorf("%s: Fenced opened and closed on %q, want %q:\n%s",
				c.name, strings.SplitN(got, "markdown", 2)[0], c.want, got)
		}

		// The fence is the one thing added: what was handed in comes back
		// whole, or a prompt is quoting something nobody said.
		if !strings.Contains(got, strings.TrimSpace(c.text)) {
			t.Errorf("%s: Fenced lost what it was given:\n%s", c.name, got)
		}
	}
}
