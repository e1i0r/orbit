package engine

// What is asked of a model, and what is asked back: the two live beside the
// Request they travel on, because both callers that write one — a phase of a
// task and the cockpit's supervisor — are asking the same models for answers
// the same panes draw, and two copies of that agreement drift.

import "strings"

// AnswerContract is how every model Orbit runs is asked to write its answer,
// and callers put it last in the prompt because the last thing said is the
// thing a model holds on to.
//
// The rules are not a house style: they are what the panes that draw the
// answer can lay out. An answer is rendered as Markdown in an eighty-column
// terminal, and a table or a block of HTML is a shape that has nowhere to go
// in one.
const AnswerContract = "## How to answer\n\n" +
	"Write the answer in Markdown, for a terminal that renders it:\n\n" +
	"- Open with one to three sentences saying what happened, before any detail.\n" +
	"- Head sections with `##`, and go no deeper than `###`.\n" +
	"- Write lists as `-` bullets.\n" +
	"- Put code in fenced blocks, and name the language on the fence.\n" +
	"- No tables and no HTML: neither can be laid out in a terminal.\n"

// Fenced is text in a code fence long enough to survive the fences already
// inside it.
//
// Everything quoted into a prompt — what the phase before said, the thread so
// far — was written to this same contract, headings and fences and all. Three
// backticks around an answer that contains three backticks close on the
// answer's own fence, and everything past that point reads as prose of the
// prompt; a heading of the answer set loose reads as a section of the prompt.
func Fenced(text string) string {
	fence := "```"
	for strings.Contains(text, fence) {
		fence += "`"
	}

	return fence + "markdown\n" + strings.TrimSpace(text) + "\n" + fence
}
